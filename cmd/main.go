package main

import (
	commands "github.com/checkmarxDev/ast-cli/commands"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	cfg := LoadConfig()
	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"logLevel": cfg.LogLevel,
		}).Fatal("Failed to parse log level")
	}

	log.SetLevel(level)
	log.SetOutput(os.Stdout)

	log.WithFields(log.Fields{
		"configuration": *cfg,
	}).Debug("Starting scans service with configuration")

	log.Info("Trying to Start Scans service")
	commands.Execute()
}
