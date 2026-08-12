package realtimeengine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestIsFeatureFlagEnabled_Success(t *testing.T) {
	mock.FFErr = nil //nolint:gocritic // resetting shared mock package state between tests
	defer func() { mock.FFErr = nil }() //nolint:gocritic // resetting shared mock package state between tests
	mock.Flag.Name = "SOME_FLAG"
	mock.Flag.Status = true

	enabled, err := IsFeatureFlagEnabled(&mock.FeatureFlagsMockWrapper{}, "SOME_FLAG")
	assert.NoError(t, err)
	assert.True(t, enabled)
}

func TestIsFeatureFlagEnabled_WrapperError_ReturnsWrappedError(t *testing.T) {
	mock.FFErr = errors.New("feature flag lookup failed") //nolint:gocritic // resetting shared mock package state between tests
	defer func() { mock.FFErr = nil }() //nolint:gocritic // resetting shared mock package state between tests

	enabled, err := IsFeatureFlagEnabled(&mock.FeatureFlagsMockWrapper{}, "SOME_FLAG")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get feature flag")
	assert.False(t, enabled)
}

func TestEnsureLicense_NilWrapper_ReturnsError(t *testing.T) {
	err := EnsureLicense(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT wrapper is not initialized")
}

func TestEnsureLicense_AtLeastOneEngineAllowed_ReturnsNil(t *testing.T) {
	jwtWrapper := &mock.JWTMockWrapper{
		CustomIsAllowedEngine: func(engine string) (bool, error) {
			return true, nil
		},
	}
	err := EnsureLicense(jwtWrapper)
	assert.NoError(t, err)
}

func TestEnsureLicense_NoEngineAllowed_ReturnsMissingLicenseError(t *testing.T) {
	jwtWrapper := &mock.JWTMockWrapper{
		CustomIsAllowedEngine: func(engine string) (bool, error) {
			return false, nil
		},
	}
	err := EnsureLicense(jwtWrapper)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errorconstants.ErrMissingAIFeatureLicense)
}

func TestEnsureLicense_WrapperError_ReturnsWrappedError(t *testing.T) {
	jwtWrapper := &mock.JWTMockWrapper{
		CustomIsAllowedEngine: func(engine string) (bool, error) {
			return false, errors.New("engine check failed")
		},
	}
	err := EnsureLicense(jwtWrapper)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check CheckmarxOneAssistType engine allowance")
}

func TestValidateFilePath_ExistingFile_ReturnsNil(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "existing-file.txt")
	err := os.WriteFile(filePath, []byte("content"), 0600)
	assert.NoError(t, err)

	err = ValidateFilePath(filePath)
	assert.NoError(t, err)
}

func TestValidateFilePath_NonExistentFile_ReturnsError(t *testing.T) {
	err := ValidateFilePath(filepath.Join(t.TempDir(), "nonexistent-file.txt"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file does not exist")
}

func TestValidateFilePath_Directory_ReturnsError(t *testing.T) {
	err := ValidateFilePath(t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is a directory")
}
