package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/logger"
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

	logger.PrintfIfVerbose("📤 Sending %d packages to realtime scanner API", len(packages.Packages))
	if len(packages.Packages) <= 10 {
		// Log all packages if few enough
		for i, pkg := range packages.Packages {
			logger.PrintfIfVerbose("  [%d] %s:%s@%s", i+1, pkg.PackageManager, pkg.PackageName, pkg.Version)
		}
	} else {
		// Log first and last few packages for large requests
		for i := 0; i < 3 && i < len(packages.Packages); i++ {
			pkg := packages.Packages[i]
			logger.PrintfIfVerbose("  [%d] %s:%s@%s", i+1, pkg.PackageManager, pkg.PackageName, pkg.Version)
		}
		logger.PrintfIfVerbose("  ... [%d more packages] ...", len(packages.Packages)-6)
		for i := len(packages.Packages) - 3; i < len(packages.Packages); i++ {
			pkg := packages.Packages[i]
			logger.PrintfIfVerbose("  [%d] %s:%s@%s", i+1, pkg.PackageManager, pkg.PackageName, pkg.Version)
		}
	}

	fn := func() (*http.Response, error) {
		return SendHTTPRequest(http.MethodPost, fmt.Sprint(r.path, "/scan/packages"), bytes.NewBuffer(jsonBytes), true, clientTimeout)
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

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		// Read response body for error details
		body, _ := io.ReadAll(resp.Body)
		logger.Printf("❌ API error (status %d): %s", resp.StatusCode, string(body))
		return nil, errors.Errorf("Failed to scan packages, status code: %s", resp.Status)
	}

	decoder := json.NewDecoder(resp.Body)
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
