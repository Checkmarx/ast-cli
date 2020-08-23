package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	resultsBfl "github.com/checkmarxDev/sast-results/pkg/web/path/bfl"

	"github.com/pkg/errors"
)

const (
	failedToParseBFL = "Failed to parse BFL"
)

type BFLHTTPWrapper struct {
	path string
}

func NewHTTPBFLWrapper(path string) BFLWrapper {
	return &BFLHTTPWrapper{
		path: path,
	}
}

func (b *BFLHTTPWrapper) GetByScanID(params map[string]string) (*resultsBfl.Forest, *resultsHelpers.WebError, error) {
	fmt.Println("REQ", b.path, params)
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, b.path, params, nil)
	if err != nil {
		return nil, nil, err
	}

	fmt.Println("Head", resp.Request.Header.Get("Authorization"))
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := resultsHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseBFL)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := resultsBfl.Forest{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseBFL)
		}
		return &model, nil, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("scan not found")
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
