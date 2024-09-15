package services

import (
	"fmt"
	"testing"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestCreateASCAScanRequest_DefaultAgent_Success(t *testing.T) {
	ASCAParams := ASCAScanParams{
		FilePath:          "data/python-vul-file.py",
		ASCAUpdateVersion: false,
		IsDefaultAgent:    true,
	}
	wrapperParams := ASCAWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		ASCAWrapper:         mock.NewASCAMockWrapper(1234),
	}
	sr, err := CreateASCAScanRequest(ASCAParams, wrapperParams)
	if err != nil {
		t.Fatalf("Failed to create ASCA scan request: %v", err)
	}
	if sr == nil {
		t.Fatalf("Failed to create ASCA scan request: %v", err)
	}
	fmt.Println(sr)
}

func TestCreateASCAScanRequest_DefaultAgentAndLatestVersionFlag_Success(t *testing.T) {
	ASCAParams := ASCAScanParams{
		FilePath:          "data/python-vul-file.py",
		ASCAUpdateVersion: true,
		IsDefaultAgent:    true,
	}
	wrapperParams := ASCAWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		ASCAWrapper:         mock.NewASCAMockWrapper(1234),
	}
	sr, err := CreateASCAScanRequest(ASCAParams, wrapperParams)
	if err != nil {
		t.Fatalf("Failed to create ASCA scan request: %v", err)
	}
	if sr == nil {
		t.Fatalf("Failed to create ASCA scan request: %v", err)
	}
	fmt.Println(sr)
}

func TestCreateASCAScanRequest_SpecialAgentAndNoLicense_Fail(t *testing.T) {
	specialErrorPort := 1
	ASCAParams := ASCAScanParams{
		FilePath:          "data/python-vul-file.py",
		ASCAUpdateVersion: true,
		IsDefaultAgent:    false,
	}
	wrapperParams := ASCAWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{AIEnabled: mock.AIProtectionDisabled},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		ASCAWrapper:         &mock.ASCAMockWrapper{Port: specialErrorPort},
	}
	_, err := CreateASCAScanRequest(ASCAParams, wrapperParams)
	assert.ErrorContains(t, err, errorconstants.NoASCALicense)
}

func TestCreateASCAScanRequest_EngineRunningAndSpecialAgentAndNoLicense_Fail(t *testing.T) {
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}

	ASCAParams := ASCAScanParams{
		FilePath:          "data/python-vul-file.py",
		ASCAUpdateVersion: true,
		IsDefaultAgent:    false,
	}

	wrapperParams := ASCAWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		ASCAWrapper:         grpcs.NewASCAGrpcWrapper(port),
	}
	err = manageASCAInstallation(ASCAParams, wrapperParams)
	assert.Nil(t, err)

	err = ensureASCAServiceRunning(wrapperParams, ASCAParams)
	assert.Nil(t, err)
	assert.Nil(t, wrapperParams.ASCAWrapper.HealthCheck())

	wrapperParams.JwtWrapper = &mock.JWTMockWrapper{AIEnabled: mock.AIProtectionDisabled}

	err = manageASCAInstallation(ASCAParams, wrapperParams)
	assert.ErrorContains(t, err, errorconstants.NoASCALicense)
	assert.NotNil(t, wrapperParams.ASCAWrapper.HealthCheck())
}

func TestCreateASCAScanRequest_EngineRunningAndDefaultAgentAndNoLicense_Success(t *testing.T) {
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}

	ASCAParams := ASCAScanParams{
		FilePath:          "data/python-vul-file.py",
		ASCAUpdateVersion: true,
		IsDefaultAgent:    true,
	}

	wrapperParams := ASCAWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		ASCAWrapper:         grpcs.NewASCAGrpcWrapper(port),
	}
	err = manageASCAInstallation(ASCAParams, wrapperParams)
	assert.Nil(t, err)

	wrapperParams.JwtWrapper = &mock.JWTMockWrapper{AIEnabled: mock.AIProtectionDisabled}

	err = ensureASCAServiceRunning(wrapperParams, ASCAParams)
	assert.Nil(t, err)
	assert.Nil(t, wrapperParams.ASCAWrapper.HealthCheck())

	err = manageASCAInstallation(ASCAParams, wrapperParams)
	assert.Nil(t, err)
	assert.Nil(t, wrapperParams.ASCAWrapper.HealthCheck())
	_ = wrapperParams.ASCAWrapper.ShutDown()
}
