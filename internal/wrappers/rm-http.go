package wrappers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

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
		return nil, errors.Wrap(err, "failed get stats")
	}
	cc := rm.CounterCollection{}
	err = json.Unmarshal(data, &cc)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshal stats")
	}
	return cc.Events, err
}

func (s *sastrmHTTPWrapper) GetScans() ([]*rm.Scan, error) {
	data, err := readData(s.url+"/scans", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed get scans")
	}
	sp := rm.ScansCollection{}
	err = json.Unmarshal(data, &sp)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshal scans")
	}
	return sp.Scans, err
}

func readData(url string, params map[string]string) ([]byte, error) {
	resp, err := SendHTTPRequestWithQueryParams("GET", url, params, nil)
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
	data, err := readData(s.url+"/engines", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed get engines")
	}
	wp := rm.EnginesCollection{}
	err = json.Unmarshal(data, &wp)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshal engines")
	}
	return wp.Engines, err
}

func NewSastRmHTTPWrapper(url string) SastRmWrapper {
	return &sastrmHTTPWrapper{
		url:         url,
		contentType: "application/json",
	}
}
