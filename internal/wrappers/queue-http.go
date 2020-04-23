package wrappers

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	rm "github.com/checkmarxDev/sast-rm/pkg/api/v1/rest"
)

type SastRmWrapper interface {
	GetScans() ([]*rm.Scan, error)
	GetWorkers() ([]*rm.Worker, error)
}

type sastrmHTTPWrapper struct {
	url         string
	contentType string
}

func (s *sastrmHTTPWrapper) GetScans() ([]*rm.Scan, error) {
	data, err := readData(s.url + "/scans")
	if err != nil {
		return nil, err
	}
	sp := rm.ScansCollection{}
	json.Unmarshal(data, &sp)
	return sp.Scans, err
}

func readData(url string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *sastrmHTTPWrapper) GetWorkers() ([]*rm.Worker, error) {
	data, err := readData(s.url + "/workers")
	if err != nil {
		return nil, err
	}
	wp := rm.WorkersCollection{}
	json.Unmarshal(data, &wp)
	return wp.Workers, err
}

func NewSastRmHTTPWrapper(url string) SastRmWrapper {
	return &sastrmHTTPWrapper{
		url:         url,
		contentType: "application/json",
	}
}
