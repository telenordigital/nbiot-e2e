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
	}
	_, err := flags.Parse(&opts)
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	m, err := NewMonitor(opts.InactivityTimeout)
	if err != nil {
		log.Fatal(err)
	}

	go m.MonitorDevices()

	for {
		m.ReceiveDeviceMessages(opts.CollectionID)
		time.Sleep(5 * time.Second)
	}
}
