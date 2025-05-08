package wrappers

import (
	"crypto/rand"
	"math/big"
	_ "time"
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

func (r RealtimeScannerHTTPWrapper) Scan(packages []OssPackageRequest) (*OssPackageResponse, error) {
	//clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	//jsonBytes, err := json.Marshal(packages)
	//if err != nil {
	//	return nil, err
	//}
	//
	//fn := func() (*http.Response, error) {
	//	return SendHTTPRequest(http.MethodPost, r.path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	//}
	//resp, err := retryHTTPRequest(fn, retryAttempts, retryDelay*time.Millisecond)
	//if err != nil {
	//	return nil, err
	//}
	//defer func() {
	//	if err == nil {
	//		_ = resp.Body.Close()
	//	}
	//}()
	//decoder := json.NewDecoder(resp.Body)
	//switch resp.StatusCode {
	//case http.StatusBadRequest, http.StatusInternalServerError:
	//	return nil, errors.Errorf("Failed to scan packages, status code: %d", resp.Status)
	//}
	//var model OssPackageResponse
	//err = decoder.Decode(&model)
	//if err != nil {
	//	return nil, errors.Wrapf(err, "failed to parse scan result")
	//}
	return generateMockResponse(packages), nil
}

func generateMockResponse(packages []OssPackageRequest) *OssPackageResponse {
	var response OssPackageResponse
	for _, pkg := range packages {
		response.Packages = append(response.Packages, OssResults{
			PackageManager: pkg.PackageManager,
			PackageName:    pkg.PackageName,
			Version:        pkg.Version,
			Status:         getRandomStatus(),
		})
	}
	return &response
}

func getRandomStatus() string {
	statuses := []string{"OK", "Malicious", "Unknown"}
	// Randomly select a status from the list
	randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(statuses))))
	if err != nil {
		return "OK" // Fallback to "OK" in case of error
	}
	return statuses[randomIndex.Int64()]
}
