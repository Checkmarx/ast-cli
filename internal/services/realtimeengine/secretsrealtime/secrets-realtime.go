package secretsrealtime

import (
	"encoding/json"
	"fmt"
	"os"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"

	"github.com/checkmarx/2ms/v3/lib/reporting"
	"github.com/checkmarx/2ms/v3/lib/secrets"
	scanner "github.com/checkmarx/2ms/v3/pkg"

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
	genericAPIKey    = "generic-api-key"
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

func filterIgnoredSecrets(results []SecretsRealtimeResult, ignoreMap map[string]bool) []SecretsRealtimeResult {
	filtered := make([]SecretsRealtimeResult, 0, len(results))
	for _, r := range results {
		key := fmt.Sprintf("%s_%s_%s", r.Title, r.FilePath, r.SecretValue)
		if !ignoreMap[key] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func buildIgnoreMap(ignored []IgnoredSecret) map[string]bool {
	m := make(map[string]bool)
	for _, s := range ignored {
		key := fmt.Sprintf("%s_%s_%s", s.Title, s.FilePath, s.SecretValue)
		m[key] = true
	}
	return m
}

func loadIgnoredSecrets(path string) ([]IgnoredSecret, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ignored []IgnoredSecret
	err = json.Unmarshal(data, &ignored)
	if err != nil {
		return nil, err
	}
	return ignored, nil
}

func (s *SecretsRealtimeService) RunSecretsRealtimeScan(filePath, ignoredFilePath string) ([]SecretsRealtimeResult, error) {
	if filePath == "" {
		return nil, errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineFilePathRequired).Error()
	}

	if enabled, err := realtimeengine.IsFeatureFlagEnabled(s.FeatureFlagWrapper, wrappers.OssRealtimeEnabled); err != nil || !enabled {
		logger.PrintfIfVerbose("Failed to print OSS Realtime scan results: %v", err)
		return nil, errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineNotAvailable).Error()
	}

	if err := realtimeengine.ValidateFilePath(filePath); err != nil {
		logger.PrintfIfVerbose("Failed to read file %s: %v", filePath, err)
		return nil, errorconstants.NewRealtimeEngineError("failed to read file").Error()
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

	results := convertToSecretsRealtimeResult(report)
	resultsPerLineMap := createResultsPerLocationMap(results)
	results = filterGenericAPIKeyVulIfNeeded(results, resultsPerLineMap)

	if ignoredFilePath == "" {
		return results, nil
	}
	ignoredSecrets, err := loadIgnoredSecrets(ignoredFilePath)
	if err != nil {
		return nil, errorconstants.NewRealtimeEngineError("failed to load ignored secrets").Error()
	}
	ignoreMap := buildIgnoreMap(ignoredSecrets)
	results = filterIgnoredSecrets(results, ignoreMap)
	return results, nil
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
	return secretScanner.Scan([]scanner.ScanItem{item}, scanner.ScanConfig{WithValidation: true})
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
		SecretValue: secret.Value,
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

func createResultsPerLocationMap(results []SecretsRealtimeResult) map[string][]SecretsRealtimeResult {
	resultsPerLocation := make(map[string][]SecretsRealtimeResult)
	for _, result := range results {
		var locationKey string
		for _, location := range result.Locations {
			locationKey = fmt.Sprintf("%s:%d", locationKey, location.Line)
		}
		resultKey := fmt.Sprintf("%s%s", result.FilePath, locationKey)
		resultsPerLocation[resultKey] = append(resultsPerLocation[resultKey], result)
	}
	return resultsPerLocation
}

func filterGenericAPIKeyVulIfNeeded(
	results []SecretsRealtimeResult,
	resultsPerLine map[string][]SecretsRealtimeResult,
) []SecretsRealtimeResult {
	if len(results) == 0 || len(resultsPerLine) == 0 {
		return results
	}

	var filtered []SecretsRealtimeResult
	for _, entries := range resultsPerLine {
		if len(entries) <= 1 {
			filtered = append(filtered, entries...)
			continue
		}

		for i := 0; i < len(entries); i++ {
			if entries[i].Title != genericAPIKey {
				filtered = append(filtered, entries[i])
			}
		}
	}
	return filtered
}
