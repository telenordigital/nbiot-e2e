package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/telenordigital/nbiot-e2e/server/pb"
	"github.com/telenordigital/nbiot-go"
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
	nbiotLibHash  string
	e2eHash       string
	rssi          float32
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
		go m.alert(deviceID, fmt.Sprintf("Expected sequence number %d but got %d", info.sequence+1, pm.Sequence), "")
	}
	info.sequence = pm.Sequence
	info.rssi = pm.Rssi
	e2eHash := fmt.Sprintf("%x", pm.E2EHash)
	if e2eHash != info.e2eHash {
		log.Printf("New version of nbiot-e2e detected\nhttps://github.com/telenordigital/nbiot-e2e/commit/%s\n", e2eHash)
		info.e2eHash = e2eHash
	}
	nbiotLibHash := fmt.Sprintf("%x", pm.NbiotLibHash)
	if nbiotLibHash != info.nbiotLibHash {
		log.Printf("New version of ArduinoNBIoT library detected\nhttps://github.com/ExploratoryEngineering/ArduinoNBIoT/commit/%s\n", nbiotLibHash)
		info.nbiotLibHash = nbiotLibHash
	}
	info.rssi = pm.Rssi

	m.deviceInfo[deviceID] = info
}

func (m *Monitor) MonitorDevices() {
	for range time.NewTicker(5 * time.Second).C {
		m.mu.Lock()
		for id, info := range m.deviceInfo {
			if time.Since(info.lastHeardFrom) > m.inactivityTimeout {
				d := m.deviceInfo[id]
				delete(m.deviceInfo, id)
				body := fmt.Sprintf(
					`Device info for last message from device:
RSSI: %v dBm
ArduinoNBIoT commit: <a href="https://github.com/ExploratoryEngineering/ArduinoNBIoT/commit/%s">%s</a>
nbiot-e2e commit: <a href="https://github.com/telenordigital/nbiot-e2e/commit/%s">%s</a>
`, d.rssi, d.nbiotLibHash, d.nbiotLibHash, d.e2eHash, d.e2eHash)
				go m.alert(id, fmt.Sprintf("not heard from for %s", m.inactivityTimeout), body)
			}
		}
		m.mu.Unlock()
	}
}

func (m *Monitor) alert(deviceID, subject, body string) {
	log.Printf("Device %s: %s", deviceID, subject)

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

	s := fmt.Sprintf("NB-IoT e2e alert! Device %s (%q): %s", deviceID, device.Tags["name"], subject)
	b := fmt.Sprintf(`%s
<a href="https://nbiot.engineering/collection-overview/%s/devices/%s">Administer device</a>

%s

You got this e-mail because you're in the <a href="https://nbiot.engineering/team-overview">%s" team</a>`,
		s, m.collectionID, deviceID, body, team.Tags["name"])

	if m.mailer == nil {
		log.Println("No mailer configured. Logging instead.")
		log.Println("Subject:", s)
		log.Println("Body: ", b)
		return
	}
	log.Println("Emailing team members...")
	for _, member := range team.Members {
		if m.mailer != nil && member.Email != nil {

			m.mailer.Send(Mail{
				To:      *member.Email,
				Subject: s,
				Body:    b,
			})
		}
	}
}
