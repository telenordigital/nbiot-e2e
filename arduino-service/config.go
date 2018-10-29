package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
)

const configFileName = ".arduino-config.json"

type Config struct {
	Sketch  string   `json:"sketch"`
	LogDir  string   `json:"logDir"`
	GitDirs []string `json:"gitDirs"`
	Boards  []Board  `json:"boards"`
}

type Board struct {
	ArduinoFQBN string `json:"fqbn"`
	Port        string `json:"port"`
}

func readConfig() (config Config, err error) {
	configFile, err := os.Open(getFullPath(configFileName))
	if err != nil {
		return config, err
	}
	defer configFile.Close()

	bytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func getFullPath(filename string) string {
	usr, err := user.Current()
	if err != nil {
		return ""
	}
	return filepath.Join(usr.HomeDir, filename)
}
