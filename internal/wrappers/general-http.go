package wrappers

import (
	"github.com/pkg/errors"

	"encoding/json"
	"io"
	"net/http"
)

func DownloadFile(downloadURL string) (io.ReadCloser, error) {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := &WebError{}
		err = decoder.Decode(errorModel)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to decode error")
		}
		return nil, errors.Errorf("%s: CODE: %d, %s", "Failed to download file", errorModel.Code, errorModel.Message)
	case http.StatusOK:
		return resp.Body, nil
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
