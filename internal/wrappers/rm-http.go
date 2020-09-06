package wrappers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

	rm "github.com/checkmarxDev/sast-rm/pkg/api/rest"
)

type sastrmHTTPWrapper struct {
	path        string
	contentType string
}

func (s *sastrmHTTPWrapper) GetStats(r StatResolution) ([]*rm.Metric, error) {
	data, err := readData(s.path+"/stats", map[string]string{
		"resolution": string(r),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed get stats")
	}
	cc := rm.MetricsCollection{}
	err = json.Unmarshal(data, &cc)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshal stats")
	}
	return cc.Metrics, err
}

func (s *sastrmHTTPWrapper) GetScans() ([]*rm.Scan, error) {
	data, err := readData(s.path+"/scans", nil)
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
	data, err := readData(s.path+"/engines", nil)
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

func NewSastRmHTTPWrapper(path string) SastRmWrapper {
	return &sastrmHTTPWrapper{
		path:        path,
		contentType: "application/json",
	}
}
