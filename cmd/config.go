package main

import (
	"github.com/spf13/viper"
)

type Config struct {
	ScansEndpoint    int
	ProjectsEndpoint int
	ResultsEndpoint  int
}

func LoadConfig() *Config {
	viper.SetDefault("LOG_LEVEL", "DEBUG")
	viper.AutomaticEnv()

	return &Config{
		ScansEndpoint:    viper.GetInt("SCANS_ENDPOINT"),
		ProjectsEndpoint: viper.GetInt("PROJECTS_ENDPOINT"),
		ResultsEndpoint:  viper.GetInt("RESULTS_ENDPOINT"),
	}
}
