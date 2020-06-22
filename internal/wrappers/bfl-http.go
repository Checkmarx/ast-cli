package wrappers

import (
	"encoding/json"
	"net/http"

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

func (b *BFLHTTPWrapper) GetByScanID(params map[string]string) (*BFLResponseModel, *ErrorModel, error) {
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, b.path, params, nil)
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseBFL)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := BFLResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseBFL)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}
