package realtimeengine

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
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
	return nil
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
