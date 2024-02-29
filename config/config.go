package config

import (
	"encoding/json"
	"log"
	"os"
)

var Config Configuration

type Configuration struct {
	BaseURL      string
	WatchDir     string
	WatchTimeSec uint
}

func SetupConfig() {
	var raw []byte
	var err error

	if raw, err = os.ReadFile("config.json"); err != nil {
		log.Fatal("Unable to read config.json file")
	}
	if err = json.Unmarshal(raw, &Config); err != nil {
		log.Fatal("Unable to parse config.json file")
	}
}
