package main

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/tarm/serial"
)

type Arduino struct {
	board, port string
	log         *log.Logger
}

func NewArduino(board, port string, logName string) *Arduino {
	logFile, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Error: ", err)
	}
	return &Arduino{
		board: board,
		port:  port,
		log:   log.New(logFile, board+" "+port+" ", log.Ldate|log.Ltime),
	}
}

func (a *Arduino) MonitorSerial() {
	a.log.Printf("Starting serial connection to %s (%s)\n", a.port, a.board)
	serialConfig := &serial.Config{Name: a.port, Baud: 9600, ReadTimeout: time.Second * 1}

	serial, err := serial.OpenPort(serialConfig)
	if err != nil {
		a.log.Println("Error: ", err)
	}

	for {
		scanner := bufio.NewScanner(serial)
		scanner.Split(scanCRLF)

		for scanner.Scan() {
			line := scanner.Text()
			a.log.Println(line)
		}
	}
}

func (a *Arduino) Verify(sketch string) error {
	a.log.Println("Verifying " + a.board)
	return arduino(a.log, "compile", "--fqbn", a.board, sketch)
}

func (a *Arduino) Upload(sketch string) error {
	a.log.Println("Uploading to " + a.port)
	return arduino(a.log, "upload", "-p", a.port, "--fqbn", a.board, sketch)
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func scanCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte{'\r', '\n'}); i >= 0 {
		// We have a full newline-terminated line.
		return i + 2, dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

func arduino(log *log.Logger, args ...string) error {
	return exe(log, "arduino-cli", args...)
}

func exe(log *log.Logger, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Error: ", err)
	}

	if err := cmd.Start(); err != nil {
		log.Println("Error: ", err)
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		log.Println(line)
	}
	if err := cmd.Wait(); err != nil {
		log.Println("Error: ", err)
		return err
	}
	return nil
}
