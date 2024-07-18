package services

import (
	"fmt"
	"testing"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestCreateVorpalScanRequest_DefaultAgent_Success(t *testing.T) {
	vorpalParams := VorpalScanParams{
		FilePath:            "data/python-vul-file.py",
		VorpalUpdateVersion: false,
		IsDefaultAgent:      true,
	}
	wrapperParams := VorpalWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		VorpalWrapper:       mock.NewVorpalMockWrapper(1234),
	}
	sr, err := CreateVorpalScanRequest(vorpalParams, wrapperParams)
	if err != nil {
		t.Fatalf("Failed to create vorpal scan request: %v", err)
	}
	if sr == nil {
		t.Fatalf("Failed to create vorpal scan request: %v", err)
	}
	fmt.Println(sr)
}

func TestCreateVorpalScanRequest_DefaultAgentAndLatestVersionFlag_Success(t *testing.T) {
	vorpalParams := VorpalScanParams{
		FilePath:            "data/python-vul-file.py",
		VorpalUpdateVersion: true,
		IsDefaultAgent:      true,
	}
	wrapperParams := VorpalWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		VorpalWrapper:       mock.NewVorpalMockWrapper(1234),
	}
	sr, err := CreateVorpalScanRequest(vorpalParams, wrapperParams)
	if err != nil {
		t.Fatalf("Failed to create vorpal scan request: %v", err)
	}
	if sr == nil {
		t.Fatalf("Failed to create vorpal scan request: %v", err)
	}
	fmt.Println(sr)
}

func TestCreateVorpalScanRequest_SpecialAgentAndNoLicense_Fail(t *testing.T) {
	specialErrorPort := 1
	vorpalParams := VorpalScanParams{
		FilePath:            "data/python-vul-file.py",
		VorpalUpdateVersion: true,
		IsDefaultAgent:      false,
	}
	wrapperParams := VorpalWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{AIEnabled: mock.AIProtectionDisabled},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		VorpalWrapper:       &mock.VorpalMockWrapper{Port: specialErrorPort},
	}
	_, err := CreateVorpalScanRequest(vorpalParams, wrapperParams)
	assert.ErrorContains(t, err, errorconstants.NoVorpalLicense)
}

func TestCreateVorpalScanRequest_EngineRunningAndSpecialAgentAndNoLicense_Fail(t *testing.T) {
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}

	vorpalParams := VorpalScanParams{
		FilePath:            "data/python-vul-file.py",
		VorpalUpdateVersion: true,
		IsDefaultAgent:      false,
	}

	wrapperParams := VorpalWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		VorpalWrapper:       grpcs.NewVorpalGrpcWrapper(port),
	}
	err = manageVorpalInstallation(vorpalParams, wrapperParams)
	assert.Nil(t, err)

	err = ensureVorpalServiceRunning(wrapperParams, vorpalParams)
	assert.Nil(t, err)
	assert.Nil(t, wrapperParams.VorpalWrapper.HealthCheck())

	wrapperParams.JwtWrapper = &mock.JWTMockWrapper{AIEnabled: mock.AIProtectionDisabled}

	err = manageVorpalInstallation(vorpalParams, wrapperParams)
	assert.ErrorContains(t, err, errorconstants.NoVorpalLicense)
	assert.NotNil(t, wrapperParams.VorpalWrapper.HealthCheck())
}

func TestCreateVorpalScanRequest_EngineRunningAndDefaultAgentAndNoLicense_Success(t *testing.T) {
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}

	vorpalParams := VorpalScanParams{
		FilePath:            "data/python-vul-file.py",
		VorpalUpdateVersion: true,
		IsDefaultAgent:      true,
	}

	wrapperParams := VorpalWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{AIEnabled: mock.AIProtectionDisabled},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		VorpalWrapper:       grpcs.NewVorpalGrpcWrapper(port),
	}
	err = manageVorpalInstallation(vorpalParams, wrapperParams)
	assert.Nil(t, err)

	err = ensureVorpalServiceRunning(wrapperParams, vorpalParams)
	assert.Nil(t, err)
	assert.Nil(t, wrapperParams.VorpalWrapper.HealthCheck())

	err = manageVorpalInstallation(vorpalParams, wrapperParams)
	assert.Nil(t, err)
	assert.Nil(t, wrapperParams.VorpalWrapper.HealthCheck())
	_ = wrapperParams.VorpalWrapper.ShutDown()
}
