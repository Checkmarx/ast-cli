package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

func (r RealtimeScannerHTTPWrapper) ScanPackages(packages *RealtimeScannerPackageRequest) (results *RealtimeScannerPackageResponse, err error) {
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
		return nil, errors.Errorf("Failed to scan packages, status code: %s", resp.Status)
	}
	var model RealtimeScannerPackageResponse
	err = decoder.Decode(&model)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse scan result")
	}
	return &model, nil
}

// ScanImages implements the RealtimeScannerWrapper interface for containers realtime.
func (r RealtimeScannerHTTPWrapper) ScanImages(images *ContainerImageRequest) (results *ContainerImageResponse, err error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(images)
	if err != nil {
		return nil, err
	}

	fn := func() (*http.Response, error) {
		return SendHTTPRequest(
			http.MethodPost,
			fmt.Sprint(r.path, "/scan/images"),
			bytes.NewBuffer(jsonBytes),
			true,
			clientTimeout,
		)
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
	case http.StatusBadRequest, http.StatusInternalServerError, http.StatusNotFound:
		return nil, errors.Errorf("Failed to scan images, status code: %s", resp.Status)
	}
	var model ContainerImageResponse
	err = decoder.Decode(&model)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse scan result")
	}
	return &model, nil
}
