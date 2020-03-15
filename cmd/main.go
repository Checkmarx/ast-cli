package main

import (
	commands "github.com/checkmarxDev/ast-cli/internal/commands"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Fatal("Failed to load ast-cli configuration")
	}

	var level log.Level
	level, err = log.ParseLevel(cfg.LogLevel)
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
	}).Debug("Starting ast-cli service with configuration")

	log.Info("Trying to start ast-cli service")

	_ = commands.Execute()

}
