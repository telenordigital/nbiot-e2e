package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/telenordigital/nbiot-go"
)

type Monitor struct {
	collectionID      string
	inactivityTimeout time.Duration
	mailer            *Mailer
	nbiot             *nbiot.Client

	mu            sync.Mutex
	lastHeardFrom map[string]time.Time
}

func NewMonitor(collectionID string, inactivityTimeout time.Duration, mailer *Mailer) (*Monitor, error) {
	client, err := nbiot.New()
	if err != nil {
		return nil, err
	}

	return &Monitor{
		collectionID:      collectionID,
		inactivityTimeout: inactivityTimeout,
		mailer:            mailer,
		nbiot:             client,
		lastHeardFrom:     map[string]time.Time{},
	}, nil
}

func (m *Monitor) ReceiveDeviceMessages() {
	stream, err := m.nbiot.CollectionOutputStream(m.collectionID)
	if err != nil {
		log.Println(err)
		return
	}
	defer stream.Close()

	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Println("Error:", err)
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

	if m.mailer == nil {
		return
	}
	log.Println("Emailing team members...")

	device, err := m.nbiot.Device(m.collectionID, deviceID)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	collection, err := m.nbiot.Collection(m.collectionID)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	team, err := m.nbiot.Team(*collection.TeamID)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	for _, member := range team.Members {
		if m.mailer != nil && member.Email != nil {
			m.mailer.Send(Mail{
				To:      *member.Email,
				Subject: fmt.Sprintf("Device %s (%q) not heard from for %s.", deviceID, device.Tags["name"], m.inactivityTimeout),
				Body:    fmt.Sprintf(`<a href="https://nbiot.engineering/collections/%s/devices/%s">Click here</a> to administer this device.`, m.collectionID, deviceID),
			})
		}
	}
}
