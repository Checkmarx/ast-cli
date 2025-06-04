package secretsrealtime

import (
	"fmt"
	"os"

	twoms "github.com/checkmarx/2ms/pkg"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type SecretsRealtimeService struct {
	JwtWrapper             wrappers.JWTWrapper
	FeatureFlagWrapper     wrappers.FeatureFlagsWrapper
	RealtimeScannerWrapper wrappers.RealtimeScannerWrapper
}

// NewOssRealtimeService creates a new OssRealtimeService.
func NewSecretsRealtimeService(
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper,
	realtimeScannerWrapper wrappers.RealtimeScannerWrapper,
) *SecretsRealtimeService {
	return &SecretsRealtimeService{
		JwtWrapper:             jwtWrapper,
		FeatureFlagWrapper:     featureFlagWrapper,
		RealtimeScannerWrapper: realtimeScannerWrapper,
	}
}

// RunOssRealtimeScan performs an OSS real-time scan on the given manifest file.
func (o *SecretsRealtimeService) RunSecretsRealtimeScan(filePath string) (*string, error) {
	scanner := twoms.NewScanner()
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil
	}
	stringContent := string(content)
	items := twoms.ScanItem{
		// convert byte to strring
		Content: &stringContent,
		ID:      "test-id",
		Source:  "test-source",
	}
	scan, err := scanner.Scan([]twoms.ScanItem{
		items,
	}, twoms.ScanConfig{
		IgnoreResultIds: []string{"test-id"},
	})
	if err != nil {
		return nil, nil
	}
	fmt.Println(scan)
	return nil, nil
}
