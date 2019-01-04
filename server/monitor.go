package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/golang/protobuf/proto"
	"github.com/telenordigital/nbiot-e2e/server/pb"
	"github.com/telenordigital/nbiot-go"
)

type Monitor struct {
	collectionID      string
	inactivityTimeout time.Duration
	mailer            *Mailer
	nbiot             *nbiot.Client
	slackURL          string

	mu         sync.Mutex
	deviceInfo map[string]*deviceInfo
}

type deviceInfo struct {
	inAlertState  bool
	lastHeardFrom time.Time
	sequence      uint32
	nbiotLibHash  uint32
	e2eHash       uint32
	rssi          float32
}

func NewMonitor(collectionID string, inactivityTimeout time.Duration, mailer *Mailer, slackURL string) (*Monitor, error) {
	client, err := nbiot.New()
	if err != nil {
		return nil, err
	}

	collection, err := client.Collection(collectionID)
	if err != nil {
		log.Fatalln("Error reading collection:", err)
	}

	team, err := client.Team(*collection.TeamID)
	if err != nil {
		log.Fatalln("Error reading team:", err)
	}

	log.Printf(`Starting monitor for collection "%s" and team "%s"`, collection.Tags["name"], team.Tags["name"])
	emailCount := 0
	for _, member := range team.Members {
		if member.Email != nil {
			emailCount++
		}
	}
	if emailCount == 0 {
		log.Fatalln("No team members with an e-mail address")
	} else {
		log.Println("Number of e-mails found in the team:", emailCount)
	}

	if slackURL == "" {
		log.Println("WARNING: no Slack webhook URL specified. Slack alerts disabled.")
	}

	return &Monitor{
		collectionID:      collectionID,
		inactivityTimeout: inactivityTimeout,
		mailer:            mailer,
		slackURL:          slackURL,
		nbiot:             client,
		deviceInfo:        map[string]*deviceInfo{},
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
	log.Printf("Received ping message from device %s %#v", deviceID, pm)

	m.mu.Lock()
	defer m.mu.Unlock()

	info, deviceExists := m.deviceInfo[deviceID]
	if !deviceExists {
		info = &deviceInfo{}
		m.deviceInfo[deviceID] = info
	}

	info.inAlertState = false
	info.lastHeardFrom = time.Now()

	if deviceExists {
		if pm.Sequence < info.sequence {
			log.Printf("Got a sequence number %d that is smaller than the previous %d. Device restarted?\n", pm.Sequence, info.sequence)
		} else if pm.Sequence != info.sequence+1 {
			go m.alert(deviceID, fmt.Sprintf("Expected sequence number %d but got %d", info.sequence+1, pm.Sequence), "")
		}

		if pm.E2EHash != info.e2eHash {
			log.Printf("New version of nbiot-e2e detected\nhttps://github.com/telenordigital/nbiot-e2e/commit/%x\n", pm.E2EHash)
		}

		if pm.NbiotLibHash != info.nbiotLibHash {
			log.Printf("New version of ArduinoNBIoT library detected\nhttps://github.com/ExploratoryEngineering/ArduinoNBIoT/commit/%x\n", pm.NbiotLibHash)
		}
	}

	info.sequence = pm.Sequence
	info.rssi = pm.PrevRssi
	info.e2eHash = pm.E2EHash
	info.nbiotLibHash = pm.NbiotLibHash
}

func (m *Monitor) MonitorDevices() {
	m.slackInfo("NB-IoT e2e server started")

	for range time.NewTicker(5 * time.Second).C {
		m.mu.Lock()
		for id, info := range m.deviceInfo {
			if info.inAlertState {
				continue
			}
			if time.Since(info.lastHeardFrom) > m.inactivityTimeout {
				info.inAlertState = true
				body := fmt.Sprintf(
					`Device info for last message from device:
RSSI: %v dBm
ArduinoNBIoT commit: %x
nbiot-e2e commit: %x
`, info.rssi, info.nbiotLibHash, info.e2eHash)
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

	subject = fmt.Sprintf("NB-IoT e2e alert! Device %q (%s): %s", device.Tags["name"], deviceID, subject)
	go m.sendEmails(deviceID, team, subject, body)
	go m.slackAlert(deviceID, team, subject, body)

}

func (m *Monitor) sendEmails(deviceID string, team nbiot.Team, subject, body string) {
	body = fmt.Sprintf(`%s
<a href="https://nbiot.engineering/collection-overview/%s/devices/%s">Administer device</a>

%s

You got this e-mail because you're in the <a href="https://nbiot.engineering/team-overview">%s" team</a>`,
		subject,
		m.collectionID,
		deviceID,
		body,
		team.Tags["name"],
	)

	if m.mailer == nil {
		log.Println("No mailer configured. Logging instead.")
		log.Println("Subject:", subject)
		log.Println("Body: ", body)
		return
	}
	log.Println("Emailing team members...")
	for _, member := range team.Members {
		if m.mailer != nil && member.Email != nil {

			m.mailer.Send(Mail{
				To:      *member.Email,
				Subject: subject,
				Body:    body,
			})
		}
	}
}

func (m *Monitor) slackInfo(text string) error {
	payload := slack.Payload{
		Text:      text,
		Username:  "e2e",
		IconEmoji: ":robot_face:",
	}
	return m.slackSend(payload)
}

func (m *Monitor) slackAlert(deviceID string, team nbiot.Team, subject, body string) error {
	color := "danger"
	text := fmt.Sprintf("%v\n%v", subject, body)
	attachment := slack.Attachment{
		Color: &color,
		Text:  &text,
	}

	deviceURL := fmt.Sprintf("https://nbiot.engineering/collection-overview/%s/devices/%s", m.collectionID, deviceID)
	collectionURL := fmt.Sprintf("https://nbiot.engineering/collection-overview/%s/devices", m.collectionID)
	attachment.AddAction(slack.Action{Type: "button", Text: "View device", Url: deviceURL, Style: "primary"})
	attachment.AddAction(slack.Action{Type: "button", Text: "View collection", Url: collectionURL})

	payload := slack.Payload{
		Username:    "e2e",
		IconEmoji:   ":robot_face:",
		Attachments: []slack.Attachment{attachment},
	}
	return m.slackSend(payload)
}

func (m *Monitor) slackSend(payload slack.Payload) error {
	if m.slackURL == "" {
		return nil
	}
	err := slack.Send(m.slackURL, "", payload)
	if len(err) > 0 {
		return fmt.Errorf("%s", err)
	}
	return nil
}
