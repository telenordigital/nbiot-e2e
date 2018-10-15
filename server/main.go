package main

import (
	"log"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
)

func main() {
	log.SetFlags(log.Llongfile | log.Ltime)

	var opts struct {
		CollectionID      string        `long:"collection-id"      required:"true" description:"The collection of devices to monitor"`
		InactivityTimeout time.Duration `long:"inactivity-timeout" default:"30s"   description:"An alert is sent if a device is not heard from for this duration"`
		DKIMPrivateKey    string        `long:"dkim-private-key"   default:""      description:"The DKIM private key for signing emails, which must also be published in a TXT record on the domain"`
	}
	_, err := flags.Parse(&opts)
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	var mailer *Mailer
	if opts.DKIMPrivateKey != "" {
		mailer = NewMailer(opts.DKIMPrivateKey)
	}

	m, err := NewMonitor(opts.CollectionID, opts.InactivityTimeout, mailer)
	if err != nil {
		log.Fatal(err)
	}

	go m.MonitorDevices()

	for {
		m.ReceiveDeviceMessages()
		time.Sleep(5 * time.Second)
	}
}
