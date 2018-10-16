package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"log"
	"time"

	"github.com/emersion/go-dkim"
	"github.com/go-mail/mail"
)

type Mailer struct {
	dkimPrivateKey *rsa.PrivateKey
	mail           chan Mail
}

type Mail struct {
	To, Subject, Body string
}

func NewMailer(dkimPrivateKey string) *Mailer {
	privateKeyBytes, err := ioutil.ReadFile(dkimPrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	pemBlock, _ := pem.Decode(privateKeyBytes)
	privateKey, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		log.Fatal(err)
	}

	m := &Mailer{
		dkimPrivateKey: privateKey,
		mail:           make(chan Mail),
	}
	go m.run()
	return m
}

func (m *Mailer) Send(mm Mail) {
	m.mail <- mm
}

func (m *Mailer) run() {
	d := mail.NewDialer("gmail-smtp-in.l.google.com", 25, "", "")
	d.StartTLSPolicy = mail.MandatoryStartTLS

	var s mail.SendCloser
	open := false
	for {
		select {
		case mm, ok := <-m.mail:
			if !ok {
				return
			}
			if !open {
				var err error
				if s, err = d.Dial(); err != nil {
					log.Println("Error:", err)
					continue
				}
				open = true
			}
			if err := m.send(s, mm); err != nil {
				log.Println("Error:", err)
			}
		case <-time.After(30 * time.Second):
			if open {
				if err := s.Close(); err != nil {
					log.Println("Error:", err)
				}
				open = false
			}
		}
	}
}

func (m *Mailer) send(sender mail.Sender, mm Mail) error {
	const from = "no-reply@nbiot.engineering"

	msg := mail.NewMessage()
	msg.SetAddressHeader("From", from, "NB-IoT end to end")
	msg.SetHeader("To", mm.To)
	msg.SetHeader("Subject", mm.Subject)
	msg.SetBody("text/html", mm.Body)

	options := &dkim.SignOptions{
		Domain:   "aviary.services",
		Selector: "main",
		Signer:   m.dkimPrivateKey,
	}

	var unsigned bytes.Buffer
	if _, err := msg.WriteTo(&unsigned); err != nil {
		return err
	}
	var signed bytes.Buffer
	if err := dkim.Sign(&signed, &unsigned, options); err != nil {
		return err
	}

	return sender.Send(from, []string{mm.To}, &signed)
}
