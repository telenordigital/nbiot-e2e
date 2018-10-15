package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/telenordigital/nbiot-go"

	"github.com/telenordigital/nbiot-e2e/server/pb"
)

type Monitor struct {
	collectionID      string
	inactivityTimeout time.Duration
	mailer            *Mailer
	nbiot             *nbiot.Client

	mu         sync.Mutex
	deviceInfo map[string]deviceInfo
}

type deviceInfo struct {
	lastHeardFrom time.Time
	sequence      uint32
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
		deviceInfo:        map[string]deviceInfo{},
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

		var message pb.Message
		if err := proto.Unmarshal(msg.Payload, &message); err != nil {
			log.Println("Error:", err)
			continue
		}

		if pm := message.GetPingMessage(); pm != nil {
			m.handlePingMessage(*msg.Device.DeviceID, *pm)
		}
	}
}

func (m *Monitor) handlePingMessage(deviceID string, pm pb.PingMessage) {
	log.Printf("Received ping message %#v", pm)

	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.deviceInfo[deviceID]
	info.lastHeardFrom = time.Now()
	if ok && pm.Sequence != info.sequence+1 {
		go m.alert(deviceID, fmt.Sprintf("expected sequence number %d but got %d", info.sequence+1, pm.Sequence))
	}
	info.sequence = pm.Sequence
	m.deviceInfo[deviceID] = info
}

func (m *Monitor) MonitorDevices() {
	for range time.NewTicker(5 * time.Second).C {
		m.mu.Lock()
		for id, info := range m.deviceInfo {
			if time.Since(info.lastHeardFrom) > m.inactivityTimeout {
				delete(m.deviceInfo, id)
				go m.alert(id, fmt.Sprintf("not heard from for %s", m.inactivityTimeout))
			}
		}
		m.mu.Unlock()
	}
}

func (m *Monitor) alert(deviceID, subject string) {
	log.Printf("Device %s: %s", deviceID, subject)

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
				Subject: fmt.Sprintf("Device %s (%q): %s", deviceID, device.Tags["name"], subject),
				Body:    fmt.Sprintf(`<a href="https://nbiot.engineering/collections/%s/devices/%s">Click here</a> to administer this device.`, m.collectionID, deviceID),
			})
		}
	}
}
