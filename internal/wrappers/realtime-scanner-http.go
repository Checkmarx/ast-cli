package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	_ "time"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type RealtimeScannerHTTPWrapper struct {
	path               string
	jwtWrapper         JWTWrapper
	featureFlagWrapper FeatureFlagsWrapper
}

func NewRealtimeScannerHTTPWrapper(path string, jwtWrapper JWTWrapper, featureFlagWrapper FeatureFlagsWrapper) *RealtimeScannerHTTPWrapper {
	return &RealtimeScannerHTTPWrapper{
		path:               path,
		jwtWrapper:         jwtWrapper,
		featureFlagWrapper: featureFlagWrapper,
	}
}

func (r RealtimeScannerHTTPWrapper) Scan(packages *OssPackageRequest) (*OssPackageResponse, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(packages)
	if err != nil {
		return nil, err
	}

	fn := func() (*http.Response, error) {
		return SendHTTPRequest(http.MethodPost, fmt.Sprint(r.path, "/analyze-manifest"), bytes.NewBuffer(jsonBytes), true, clientTimeout)
	}
	resp, err := retryHTTPRequest(fn, retryAttempts, retryDelay*time.Millisecond)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	decoder := json.NewDecoder(resp.Body)
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		return nil, errors.Errorf("Failed to scan packages, status code: %d", resp.Status)
	}
	var model OssPackageResponse
	err = decoder.Decode(&model)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse scan result")
	}
	return &model, nil
}
