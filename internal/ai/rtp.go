// Package ai bridges Asterisk calls to OpenAI realtime voice agents.
package ai

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"time"
)

const (
	frameBytes   = 160 // 20 ms of 8 kHz G.711 μ-law
	frameDur     = 20 * time.Millisecond
	payloadPCMU  = 0
	ulawSilence  = 0xFF
	maxBuffered  = 8000 * 60 // one minute of audio
	rtpHeaderLen = 12
)

// RTPEndpoint exchanges G.711 μ-law audio with an Asterisk ExternalMedia
// channel. The remote address is learned from the first inbound packet.
type RTPEndpoint struct {
	conn *net.UDPConn

	mu     sync.Mutex
	remote *net.UDPAddr
	out    []byte
	sent   int64 // bytes of assistant audio actually played out

	seq  uint16
	ts   uint32
	ssrc uint32
}

// ListenRTP binds a UDP socket on host:port.
func ListenRTP(host string, port int) (*RTPEndpoint, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return &RTPEndpoint{conn: conn, seq: uint16(rand.IntN(65535)), ts: rand.Uint32(), ssrc: rand.Uint32()}, nil
}

// LocalPort returns the bound UDP port.
func (r *RTPEndpoint) LocalPort() int { return r.conn.LocalAddr().(*net.UDPAddr).Port }

// Close releases the socket.
func (r *RTPEndpoint) Close() error { return r.conn.Close() }

// ReadLoop delivers inbound audio payloads to onAudio until ctx ends.
func (r *RTPEndpoint) ReadLoop(ctx context.Context, onAudio func([]byte)) error {
	buf := make([]byte, 2048)
	go func() {
		<-ctx.Done()
		_ = r.conn.SetReadDeadline(time.Now())
	}()
	for {
		n, from, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return err
		}
		payload, ok := parseRTP(buf[:n])
		if !ok {
			continue
		}
		r.mu.Lock()
		if r.remote == nil {
			r.remote = from
		}
		r.mu.Unlock()
		onAudio(append([]byte(nil), payload...))
	}
}

func parseRTP(pkt []byte) ([]byte, bool) {
	if len(pkt) < rtpHeaderLen || pkt[0]>>6 != 2 {
		return nil, false
	}
	off := rtpHeaderLen + int(pkt[0]&0x0F)*4
	if pkt[0]&0x10 != 0 { // header extension
		if len(pkt) < off+4 {
			return nil, false
		}
		off += 4 + int(binary.BigEndian.Uint16(pkt[off+2:]))*4
	}
	end := len(pkt)
	if pkt[0]&0x20 != 0 && end > 0 { // padding
		end -= int(pkt[end-1])
	}
	if off > end {
		return nil, false
	}
	return pkt[off:end], true
}

// Enqueue appends assistant audio for paced playout.
func (r *RTPEndpoint) Enqueue(audio []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.out)+len(audio) > maxBuffered {
		return
	}
	r.out = append(r.out, audio...)
}

// Flush drops pending assistant audio when the caller barges in.
func (r *RTPEndpoint) Flush() {
	r.mu.Lock()
	r.out = r.out[:0]
	r.mu.Unlock()
}

// Pending returns the buffered audio duration.
func (r *RTPEndpoint) Pending() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return time.Duration(len(r.out)) * time.Second / 8000
}

// PlayedMS returns total assistant audio played, in milliseconds.
func (r *RTPEndpoint) PlayedMS() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sent / 8
}

// WriteLoop paces buffered audio out as 20 ms RTP frames until ctx ends.
func (r *RTPEndpoint) WriteLoop(ctx context.Context) {
	ticker := time.NewTicker(frameDur)
	defer ticker.Stop()
	pkt := make([]byte, rtpHeaderLen+frameBytes)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		r.mu.Lock()
		remote := r.remote
		if remote == nil || len(r.out) == 0 {
			r.mu.Unlock()
			continue
		}
		n := copy(pkt[rtpHeaderLen:], r.out)
		for i := rtpHeaderLen + n; i < len(pkt); i++ {
			pkt[i] = ulawSilence
		}
		r.out = r.out[n:]
		r.sent += int64(n)
		r.seq++
		r.ts += frameBytes
		pkt[0], pkt[1] = 0x80, payloadPCMU
		binary.BigEndian.PutUint16(pkt[2:], r.seq)
		binary.BigEndian.PutUint32(pkt[4:], r.ts)
		binary.BigEndian.PutUint32(pkt[8:], r.ssrc)
		r.mu.Unlock()
		_, _ = r.conn.WriteToUDP(pkt, remote)
	}
}
