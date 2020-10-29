package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

	rm "github.com/checkmarxDev/sast-rm/pkg/api/rest"
)

type sastrmHTTPWrapper struct {
	path        string
	contentType string
}

func (s *sastrmHTTPWrapper) AddPool(description string) (pool *rm.Pool, err error) {
	pool = &rm.Pool{
		Description: "description",
	}
	err = postData(s.path+"/pools", pool, nil)
	return pool, errors.Wrap(err, "failed to create pool")
}

func (s *sastrmHTTPWrapper) DeletePool(id string) error {
	path := fmt.Sprintf("%s/pools/{%s}", s.path, id)
	resp, err := SendHTTPRequestWithQueryParams(http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (s *sastrmHTTPWrapper) GetPools() ([]*rm.Pool, error) {
	data, err := readData(s.path+"/pools", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed get pools")
	}
	pc := rm.PoolsCollection{}
	err = json.Unmarshal(data, &pc)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshal pools")
	}
	return pc.Pools, err
}

func (s *sastrmHTTPWrapper) GetPoolEngines(id string) (engines []string, err error) {
	url := fmt.Sprintf("%s/pools/{%s}/engines", s.path, id)
	err = read(url, engines)
	return engines, errors.Wrap(err, "failed get pool engines")
}

func (s *sastrmHTTPWrapper) GetPoolProjects(id string) (projects []string, err error) {
	url := fmt.Sprintf("%s/pools/{%s}/projects", s.path, id)
	err = read(url, projects)
	return projects, errors.Wrap(err, "failed get pool projects")
}

func (s *sastrmHTTPWrapper) GetPoolEngineTags(id string) (tags map[string]string, err error) {
	url := fmt.Sprintf("%s/pools/{%s}/engine-tags", s.path, id)
	err = read(url, tags)
	return tags, errors.Wrap(err, "failed get engine tags")
}

func (s *sastrmHTTPWrapper) GetPoolProjectTags(id string) (tags map[string]string, err error) {
	url := fmt.Sprintf("%s/pools/{%s}/project-tags", s.path, id)
	err = read(url, tags)
	return tags, errors.Wrap(err, "failed get project tags")
}

func (s *sastrmHTTPWrapper) SetPoolEngines(id string, value []string) error {
	url := fmt.Sprintf("%s/pools/{%s}/engines", s.path, id)
	err := putData(url, value, nil)
	return errors.Wrap(err, "failed to set pool engines")
}

func (s *sastrmHTTPWrapper) SetPoolProjects(id string, value []string) error {
	url := fmt.Sprintf("%s/pools/{%s}/projects", s.path, id)
	err := putData(url, value, nil)
	return errors.Wrap(err, "failed to set pool projects")
}

func (s *sastrmHTTPWrapper) SetPoolEngineTags(id string, tags map[string]string) error {
	url := fmt.Sprintf("%s/pools/{%s}/engine-tags", s.path, id)
	err := putData(url, toKeyValue(tags), nil)
	return errors.Wrap(err, "failed to set pool engine tags")
}

func toKeyValue(value map[string]string) (kvps []struct{ Key, Value string }) {
	for k, v := range value {
		kvps = append(kvps, struct{ Key, Value string }{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (s *sastrmHTTPWrapper) SetPoolProjectTags(id string, tags map[string]string) error {
	url := fmt.Sprintf("%s/pools/{%s}/project-tags", s.path, id)
	err := putData(url, toKeyValue(tags), nil)
	return errors.Wrap(err, "failed to set pool project tags")
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

func read(url string, result interface{}) error {
	data, err := readData(url, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, result)
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

func postData(url string, data interface{}, params map[string]string) error {
	requestData, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data")
	}

	resp, err := SendHTTPRequestWithQueryParams(http.MethodPost, url, params, bytes.NewReader(requestData))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.New(resp.Status)
	}
	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(responseData, &data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal data")
	}
	return resp.Body.Close()
}

func putData(url string, data interface{}, params map[string]string) error {
	requestData, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data")
	}

	resp, err := SendHTTPRequestWithQueryParams(http.MethodPut, url, params, bytes.NewReader(requestData))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return errors.New(resp.Status)
	}
	return resp.Body.Close()
}

func NewSastRmHTTPWrapper(path string) SastRmWrapper {
	return &sastrmHTTPWrapper{
		path:        path,
		contentType: "application/json",
	}
}
