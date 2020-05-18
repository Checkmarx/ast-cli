package wrappers

import (
	"net/http"

	"github.com/pkg/errors"
)

type SimpleHealthcheckWrapper struct {
	webAppURL string
}

func NewSimpleHealthcheckWrapper(astWebAppURL string) HealthcheckWrapper {
	return &SimpleHealthcheckWrapper{
		webAppURL: astWebAppURL,
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
