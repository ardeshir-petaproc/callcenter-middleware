package integration

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func waitFor(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// fakeSMTP accepts one-recipient messages and hands each DATA payload to
// the returned channel. It advertises no AUTH and no STARTTLS.
func fakeSMTP(t *testing.T) (int, <-chan string) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	got := make(chan string, 4)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go serveSMTP(c, got)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, got
}

func serveSMTP(c net.Conn, got chan<- string) {
	defer c.Close()
	r := bufio.NewReader(c)
	fmt.Fprint(c, "220 fake ESMTP\r\n")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		switch cmd := strings.ToUpper(strings.Fields(line + " x")[0]); cmd {
		case "EHLO", "HELO":
			fmt.Fprint(c, "250 fake\r\n")
		case "DATA":
			fmt.Fprint(c, "354 go ahead\r\n")
			var b strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil || l == ".\r\n" {
					break
				}
				b.WriteString(l)
			}
			got <- b.String()
			fmt.Fprint(c, "250 queued\r\n")
		case "QUIT":
			fmt.Fprint(c, "221 bye\r\n")
			return
		default:
			fmt.Fprint(c, "250 ok\r\n")
		}
	}
}
