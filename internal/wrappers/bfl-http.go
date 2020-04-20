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
	url string
}

func NewHTTPBFLWrapper(url string) BFLWrapper {
	return &BFLHTTPWrapper{
		url: url,
	}
}

func (b *BFLHTTPWrapper) GetByScanID(scanID string, limit, offset uint64) (*BFLResponseModel, *ErrorModel, error) {
	params := make(map[string]string)
	params["scan-id"] = scanID
	resp, err := SendHTTPRequestWithLimitAndOffset(http.MethodGet, b.url, params, limit, offset, nil)
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
