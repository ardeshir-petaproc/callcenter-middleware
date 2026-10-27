package ai

import "encoding/binary"

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
