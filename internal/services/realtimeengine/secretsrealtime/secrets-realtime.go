package secretsrealtime

import (
	"fmt"
	"os"

	"github.com/checkmarx/2ms/v3/lib/reporting"
	"github.com/checkmarx/2ms/v3/lib/secrets"
	scanner "github.com/checkmarx/2ms/v3/pkg"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	validSecret      = "Valid"
	invalidSecret    = "Invalid"
	unknownSecret    = "Unknown"
	criticalSeverity = "Critical"
	highSeverity     = "High"
	mediumSeverity   = "Medium"
)

type SecretsRealtimeService struct {
	JwtWrapper             wrappers.JWTWrapper
	FeatureFlagWrapper     wrappers.FeatureFlagsWrapper
	RealtimeScannerWrapper wrappers.RealtimeScannerWrapper
}

func NewSecretsRealtimeService(
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper,
) *SecretsRealtimeService {
	return &SecretsRealtimeService{
		JwtWrapper:         jwtWrapper,
		FeatureFlagWrapper: featureFlagWrapper,
	}
}

func (s *SecretsRealtimeService) RunSecretsRealtimeScan(filePath string) ([]SecretsRealtimeResult, error) {
	if filePath == "" {
		return nil, errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineFilePathRequired).Error()
	}

	if enabled, err := s.FeatureFlagWrapper.GetSpecificFlag(wrappers.OssRealtimeEnabled); err != nil || !enabled.Status {
		logger.PrintfIfVerbose("Failed to print OSS Realtime scan results: %v", err)
		return nil, errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineNotAvailable).Error()
	}

	content, err := readFile(filePath)
	if err != nil {
		logger.PrintfIfVerbose("Failed to read file %s: %v", filePath, err)
		return nil, errorconstants.NewRealtimeEngineError("failed to read file").Error()
	}

	report, err := runScan(filePath, content)
	if err != nil {
		logger.PrintfIfVerbose("Failed to run scan: %v", err)
		return nil, errorconstants.NewRealtimeEngineError("failed to run secrets scan").Error()
	}

	return convertToSecretsRealtimeResult(report), nil
}

func readFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("readFile: %w", err)
	}
	return string(data), nil
}

func runScan(source, content string) (*reporting.Report, error) {
	item := scanner.ScanItem{
		Content: &content,
		Source:  source,
	}
	secretScanner := scanner.NewScanner()
	return secretScanner.ScanWithValidation([]scanner.ScanItem{item}, scanner.ScanConfig{})
}

func convertToSecretsRealtimeResult(report *reporting.Report) []SecretsRealtimeResult {
	results := make([]SecretsRealtimeResult, 0)
	for _, resultGroup := range report.Results {
		for _, secret := range resultGroup {
			results = append(results, convertSecretToResult(secret))
		}
	}
	return results
}

func convertSecretToResult(secret *secrets.Secret) SecretsRealtimeResult {
	var locations []realtimeengine.Location
	for i := 0; i <= secret.EndLine-secret.StartLine; i++ {
		locations = append(locations, realtimeengine.Location{
			Line:       secret.StartLine + i,
			StartIndex: secret.StartColumn,
			EndIndex:   secret.EndColumn,
		})
	}

	return SecretsRealtimeResult{
		Title:       secret.RuleID,
		Description: secret.RuleDescription,
		Severity:    getSeverity(secret),
		FilePath:    secret.Source,
		Locations:   locations,
	}
}

func getSeverity(secret *secrets.Secret) string {
	switch secret.ValidationStatus {
	case validSecret:
		return criticalSeverity
	case unknownSecret:
		return highSeverity
	case invalidSecret:
		return mediumSeverity
	default:
		return highSeverity
	}
}
