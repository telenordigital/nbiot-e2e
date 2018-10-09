package main

import (
	"log"
	"sync"
	"time"

	"github.com/telenordigital/nbiot-go"
)

type Monitor struct {
	inactivityTimeout time.Duration
	nbiot             *nbiot.Client

	mu            sync.Mutex
	lastHeardFrom map[string]time.Time
}

func NewMonitor(inactivityTimeout time.Duration) (*Monitor, error) {
	client, err := nbiot.New()
	if err != nil {
		return nil, err
	}

	return &Monitor{
		inactivityTimeout: inactivityTimeout,
		nbiot:             client,
		lastHeardFrom:     map[string]time.Time{},
	}, nil
}

func (m *Monitor) ReceiveDeviceMessages(collectionID string) {
	stream, err := m.nbiot.CollectionOutputStream(collectionID)
	if err != nil {
		log.Println(err)
		return
	}
	defer stream.Close()

	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("Received message %#v", msg)

		m.mu.Lock()
		m.lastHeardFrom[*msg.Device.DeviceID] = time.Now()
		m.mu.Unlock()
	}
}

func (m *Monitor) MonitorDevices() {
	for range time.NewTicker(5 * time.Second).C {
		m.mu.Lock()
		for id, t := range m.lastHeardFrom {
			if time.Since(t) > m.inactivityTimeout {
				delete(m.lastHeardFrom, id)
				go m.alert(id)
			}
		}
		m.mu.Unlock()
	}
}

func (m *Monitor) alert(deviceID string) {
	log.Printf("Device %s not heard from for %v.", deviceID, m.inactivityTimeout)
}
