package services

import (
	"fmt"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
)

func TestCreateVorpalScanRequest(t *testing.T) {
	vorpalParams := VorpalScanParams{
		FilePath:            "/Users/benalvo/CxDev/workspace/Pheonix-workspace/CxCodeProbe/testdata/Java/samples/Cookies.java",
		VorpalUpdateVersion: false,
		IsDefaultAgent:      true,
		JwtWrapper:          &mock.JWTMockWrapper{},
		FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
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
