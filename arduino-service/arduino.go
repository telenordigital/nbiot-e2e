package main

import (
	"bufio"
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
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

func (a *Arduino) Compile(sketch string, gitHashes []gitHash) error {
	a.log.Println("Compiling " + a.board)

	args := []string{"compile", "--fqbn", a.board}
	var defines []string
	for _, gitHash := range gitHashes {
		defineName, err := repoToDefineName(gitHash.Name)
		if err != nil {
			log.Fatalln("Error:", err)
		}
		defines = append(defines, defineName+"="+gitHash.LastCommit)
	}
	if len(defines) > 0 {
		args = append(args, "--build-properties", "\"build.extra_flags=-D"+strings.Join(defines, " -D")+"\"")
	}

	args = append(args, sketch)
	a.log.Printf("Compile args: %v\n", args)
	return arduino(a.log, args...)
}

func repoToDefineName(repoName string) (string, error) {
	switch repoName {
	case "ArduinoNBIoT":
		return "NBIOT_LIB_HASH", nil
	case "nbiot-e2e":
		return "E2E_HASH", nil
	default:
		return "", errors.New("Unknown repo name")
	}
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
