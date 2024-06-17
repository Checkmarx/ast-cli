package services

import (
	"fmt"
	"testing"

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
	aiDisabled := 0
	vorpalParams := VorpalScanParams{
		FilePath:            "data/python-vul-file.py",
		VorpalUpdateVersion: true,
		IsDefaultAgent:      false,
	}
	wrapperParams := VorpalWrappersParam{
		JwtWrapper:          &mock.JWTMockWrapper{AIEnabled: aiDisabled},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		VorpalWrapper:       &mock.VorpalMockWrapper{Port: specialErrorPort},
	}
	_, err := CreateVorpalScanRequest(vorpalParams, wrapperParams)
	assert.Error(t, err)
}
