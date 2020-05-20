package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	healthcheckV1 "github.com/checkmarxDev/healthcheck/api/rest/v1"
	"github.com/pkg/errors"
)

type SimpleHealthcheckWrapper struct {
	webAppURL      string
	healthcheckURL string
}

func NewSimpleHealthcheckWrapper(astWebAppURL, healthcheckURL string) HealthcheckWrapper {
	return &SimpleHealthcheckWrapper{
		webAppURL:      astWebAppURL,
		healthcheckURL: healthcheckURL,
	}
}

func (s *SimpleHealthcheckWrapper) CheckWebAppIsUp() error {
	resp, err := SendHTTPRequest("GET", s.webAppURL, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("HTTP status code %d", resp.StatusCode)
	}
	return nil
}

func (s *SimpleHealthcheckWrapper) CheckDatabaseHealth() error {
	fmt.Println(s.healthcheckURL)
	resp, err := SendHTTPRequest("GET", s.healthcheckURL+"/database", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("HTTP status code %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(resp.Body)
	model := healthcheckV1.HealthcheckModel{}
	err = decoder.Decode(&model)
	if err != nil {
		return err
	}
	if !model.Success {
		return errors.New(model.Message)
	}

	return nil
}
