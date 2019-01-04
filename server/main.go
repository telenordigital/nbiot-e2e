package main

import (
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/jessevdk/go-flags"
)

const (
	// SlackWebhookEnvName is the name of the environment variable that can be
	// used to specify the Slack webhook URL for posting alerts
	SlackWebhookEnvName = "SLACK_WEBHOOK_URL"
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

	slackWebhookURL := os.Getenv(SlackWebhookEnvName)
	if slackWebhookURL == "" {
		log.Println("No ENV variable for slack webhook URL, trying to fetch secret from AWS")
		log.Println("AWS_REGION: ", os.Getenv("AWS_REGION"))
		slackWebhookURL, err = getParameter("nbiot-e2e-slack-webhook-url")
		if err != nil {
			log.Println("Error: ", err)
		}
		log.Println("Slack webhook: ", slackWebhookURL)
	}

	var mailer *Mailer
	if opts.DKIMPrivateKey != "" {
		mailer = NewMailer(opts.DKIMPrivateKey)
	}

	m, err := NewMonitor(opts.CollectionID, opts.InactivityTimeout, mailer, slackWebhookURL)
	if err != nil {
		log.Fatal(err)
	}

	go m.MonitorDevices()

	for {
		m.ReceiveDeviceMessages()
		time.Sleep(5 * time.Second)
	}
}

func getParameter(name string) (string, error) {
	svc := ssm.New(session.New())
	result, err := svc.GetParameter(&ssm.GetParameterInput{Name: &name})
	if err != nil {
		return "", err
	}
	return *result.Parameter.Value, nil
}
