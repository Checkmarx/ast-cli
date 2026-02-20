package realtimeengine

import (
	"os"
	"strings"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

const (
	Malicious = "malicious"
	Critical  = "critical"
	High      = "high"
	Medium    = "medium"
	Low       = "low"
)

// IsFeatureFlagEnabled checks if a specific feature flag is enabled.
func IsFeatureFlagEnabled(flagWrapper wrappers.FeatureFlagsWrapper, flagName string) (bool, error) {
	enabled, err := flagWrapper.GetSpecificFlag(flagName)
	if err != nil {
		return false, errors.Wrap(err, "failed to get feature flag")
	}
	return enabled.Status, nil
}

// EnsureLicense validates that a valid JWT wrapper is available.
func EnsureLicense(jwtWrapper wrappers.JWTWrapper) error {
	if jwtWrapper == nil {
		return errors.New("JWT wrapper is not initialized, cannot ensure license")
	}

	assistAllowed, err := jwtWrapper.IsAllowedEngine(params.CheckmarxOneAssistType)
	if err != nil {
		return errors.Wrap(err, "failed to check CheckmarxOneAssistType engine allowance")
	}

	aiAllowed, err := jwtWrapper.IsAllowedEngine(params.AIProtectionType)
	if err != nil {
		return errors.Wrap(err, "failed to check AIProtectionType engine allowance")
	}

	devAssistAllowed, err := jwtWrapper.IsAllowedEngine(params.CheckmarxDevAssistType)
	if err != nil {
		return errors.Wrap(err, "failed to check Checkmarx Developer Assist engine allowance")
	}

	if aiAllowed || assistAllowed || devAssistAllowed {
		return nil
	}
	return errors.New(errorconstants.ErrMissingAIFeatureLicense)
}

// ValidateFilePath validates that the file path exists and is accessible.
func ValidateFilePath(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.Errorf("file does not exist: %s", filePath)
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return errors.Errorf("cannot access file: %s", filePath)
	}
	if fileInfo.IsDir() {
		return errors.Errorf("path is a directory, expected a file: %s", filePath)
	}
	return nil
}

func ContainsSeverity(vulnerabilitySeverity string, severitiesThreshold []string) bool {
	for _, severity := range severitiesThreshold {
		severity = strings.TrimSpace(severity)
		if strings.EqualFold(severity, vulnerabilitySeverity) {
			return true
		}
	}
	return false
}
func IsStatusUnknownOrOK(status string) bool {
	if strings.EqualFold(status, "unknown") || strings.EqualFold(status, "ok") {
		return true
	}
	return false
}

func IsSeverityValid(severities []string) error {
	if len(severities) == 0 {
		return nil
	}
	for _, severity := range severities {
		severity = strings.TrimSpace(severity)
		severity = strings.ToLower(severity)
		if severity != Malicious && severity != Critical && severity != High && severity != Medium && severity != Low {
			return errors.Errorf("Invalid severity threshold value: %s", severity)
		}
	}
	return nil
}
