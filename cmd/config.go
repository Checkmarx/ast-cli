package main

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	scansEp    = "SCANS_ENDPOINT"
	projectsEp = "PROJECTS_ENDPOINT"
	resultsEp  = "RESULTS_ENDPOINT"
	logLevel   = "LOG_LEVEL"
)

type Config struct {
	ScansEndpoint    string
	ProjectsEndpoint string
	ResultsEndpoint  string
	LogLevel         string
}

func LoadConfig() (*Config, error) {

	viper.AutomaticEnv()
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("env")
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	viper.SetDefault(logLevel, "DEBUG")

	scans := viper.GetString(scansEp)
	if scans == "" {
		return requiredErr(scansEp)
	}
	projects := viper.GetString(projectsEp)
	if projects == "" {
		return requiredErr(projectsEp)
	}
	results := viper.GetString(resultsEp)
	if results == "" {
		return requiredErr(resultsEp)
	}
	logLevel := viper.GetString("LOG_LEVEL")
	return &Config{
		ScansEndpoint:    scans,
		ProjectsEndpoint: projects,
		ResultsEndpoint:  results,
		LogLevel:         logLevel,
	}, nil
}

func requiredErr(key string) (*Config, error) {
	return nil, errors.Errorf("%s is required", key)
}
