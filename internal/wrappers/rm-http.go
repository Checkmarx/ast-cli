package wrappers

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	rm "github.com/checkmarxDev/sast-rm/pkg/api/v1/rest"
)

type sastrmHTTPWrapper struct {
	url         string
	contentType string
}

func (s *sastrmHTTPWrapper) GetStats(m StatMetric, r StatResolution) ([]*rm.Counter, error) {
	data, err := readData(s.url+"/stats", map[string]string{
		"resolution": string(r),
		"metric":     string(m),
	})
	if err != nil {
		return nil, err
	}
	cc := rm.CounterCollection{}
	json.Unmarshal(data, &cc)
	return cc.Events, err
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

func readData(url string, params ...map[string]string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params[0] {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
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

func (s *sastrmHTTPWrapper) GetEngines() ([]*rm.Engine, error) {
	data, err := readData(s.url + "/engines")
	if err != nil {
		return nil, err
	}
	wp := rm.EnginesCollection{}
	json.Unmarshal(data, &wp)
	return wp.Engines, err
}

func NewSastRmHTTPWrapper(url string) SastRmWrapper {
	return &sastrmHTTPWrapper{
		url:         url,
		contentType: "application/json",
	}
}
