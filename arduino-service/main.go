package main

import (
	"log"
	"os"
	"path"
	"strings"

	"gopkg.in/src-d/go-git.v4"
)

type gitHash struct {
	Name       string
	LastCommit string
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	config, err := readConfig()
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := os.Stat(config.Sketch); os.IsNotExist(err) {
		log.Fatalln(err)
	}

	if stat, err := os.Stat(config.LogDir); os.IsNotExist(err) {
		log.Fatalln(err)
	} else if !stat.IsDir() {
		log.Fatalf("%s is not a directory", config.LogDir)
	}

	for _, b := range config.Boards {
		logName := path.Join(config.LogDir, strings.Replace(b.ArduinoFQBN, ":", "-", -1)+".log")
		a := NewArduino(b.ArduinoFQBN, b.Port, logName)

		if err := a.Compile(config.Sketch, gitHashes(config.GitDirs)); err != nil {
			log.Println("Error: ", err)
		} else {
			if err = a.Upload(config.Sketch); err != nil {
				log.Println("Error: ", err)
			}
		}

		log.Printf("Starting serial monitor for %s %s\n", b.ArduinoFQBN, b.Port)
		go a.MonitorSerial()
	}

	select {}
}

func gitHashes(gitDirs []string) []gitHash {
	gitHashes := make([]gitHash, len(gitDirs))
	for i, gitDir := range gitDirs {
		repo, err := git.PlainOpen(gitDir)
		if err != nil {
			log.Fatal("Error: ", err)
		}
		head, err := repo.Head()
		if err != nil {
			log.Fatal("Error: ", err)
		}
		origin, err := repo.Remote("origin")
		if err != nil {
			log.Fatal("Error: ", err)
		}
		repoName := strings.Replace(path.Base(origin.Config().URLs[0]), ".git", "", 1)
		gitHashes[i] = gitHash{repoName, head.Hash().String()[:7]}
		log.Printf("Last commit %v", gitHashes[i])
	}
	return gitHashes
}
