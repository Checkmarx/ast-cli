package services

import (
	"fmt"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
)

func TestCreateVorpalScanRequest(t *testing.T) {
	vorpalParams := VorpalScanParams{
		FilePath:            "data/python-vul-file.py",
		VorpalUpdateVersion: false,
		IsDefaultAgent:      true,
		JwtWrapper:          &mock.JWTMockWrapper{},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
		VorpalWrapper:       &mock.VorpalMockWrapper{},
	}
	sr, err := CreateVorpalScanRequest(vorpalParams)
	if err != nil {
		t.Fatalf("Failed to create vorpal scan request: %v", err)
	}
	if sr == nil {
		t.Fatalf("Failed to create vorpal scan request: %v", err)
	}
	fmt.Println(sr)
}
