// Command mail-sim is a local mailbox for trying the email connector without
// a real mail server: an in-memory IMAP server (user "username", password
// "password", folder INBOX) and an SMTP sink that prints replies. POST a
// message to the HTTP port to drop it into the inbox:
//
//	curl -XPOST localhost:1080/deliver -d 'from=ali@customer.com&subject=Invoice&body=Wrong total'
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/emersion/go-imap/backend/memory"
	imapserver "github.com/emersion/go-imap/server"
)

func main() {
	imapAddr := flag.String("imap", ":1143", "IMAP listen address")
	smtpAddr := flag.String("smtp", ":1025", "SMTP listen address")
	httpAddr := flag.String("http", ":1080", "HTTP address for POST /deliver")
	flag.Parse()

	be := memory.New()
	srv := imapserver.New(be)
	srv.Addr, srv.AllowInsecureAuth = *imapAddr, true
	go func() { log.Fatal(srv.ListenAndServe()) }()
	go serveSMTP(*smtpAddr)

	user, _ := be.Login(nil, "username", "password")
	inbox, _ := user.GetMailbox("INBOX")
	http.HandleFunc("POST /deliver", func(w http.ResponseWriter, r *http.Request) {
		from, subject := r.FormValue("from"), r.FormValue("subject")
		id := fmt.Sprintf("%d@mail-sim", time.Now().UnixNano())
		msg := fmt.Sprintf("From: %s\r\nTo: support@example.com\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s>\r\n",
			from, subject, time.Now().Format(time.RFC1123Z), id)
		if irt := r.FormValue("in_reply_to"); irt != "" {
			msg += "In-Reply-To: <" + irt + ">\r\n"
		}
		msg += "Content-Type: text/plain; charset=utf-8\r\n\r\n" + r.FormValue("body")
		if err := inbox.CreateMessage(nil, time.Now(), bytes.NewBufferString(msg)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, id)
	})
	log.Printf("mail-sim: IMAP %s, SMTP %s, deliver via POST http://%s/deliver", *imapAddr, *smtpAddr, *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

// serveSMTP accepts any message and logs it (no AUTH, no TLS).
func serveSMTP(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			r := bufio.NewReader(c)
			fmt.Fprint(c, "220 mail-sim\r\n")
			for {
				line, err := r.ReadString('\n')
				if err != nil {
					return
				}
				switch strings.ToUpper(strings.Fields(line + " x")[0]) {
				case "DATA":
					fmt.Fprint(c, "354 go ahead\r\n")
					var b strings.Builder
					for l, err := r.ReadString('\n'); err == nil && l != ".\r\n"; l, err = r.ReadString('\n') {
						b.WriteString(l)
					}
					log.Printf("SMTP message:\n%s", b.String())
					fmt.Fprint(c, "250 queued\r\n")
				case "QUIT":
					fmt.Fprint(c, "221 bye\r\n")
					return
				default:
					fmt.Fprint(c, "250 ok\r\n")
				}
			}
		}(c)
	}
}
