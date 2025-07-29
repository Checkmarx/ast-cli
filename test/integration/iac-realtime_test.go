//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/stretchr/testify/assert"
)

const maxJSONSize = 10 * 1024 * 1024

func safeJSONUnmarshal(data []byte, v interface{}) error {
	if len(data) > maxJSONSize {
		return json.Unmarshal(data[:maxJSONSize], v)
	}
	return json.Unmarshal(data, v)
}

func safeRemoveFile(path string) {
	cleanPath := filepath.Clean(path)
	if err := os.Remove(cleanPath); err != nil && !os.IsNotExist(err) && !os.IsPermission(err) {
		// Log error but don't fail the test
		return
	}
}

func TestIacRealtimeScan_TerraformFile_Success(t *testing.T) {
	configuration.LoadConfiguration()

	args := []string{
		"scan", "iac-realtime",
		flag(commonParams.SourcesFlag), "data/positive1.tf",
		flag(commonParams.IacRealtimeEngineFlag), "docker",
	}

	err, bytes := executeCommand(t, args...)

	assert.Nil(t, err, "Scanning Terraform file should not fail")

	var results []iacrealtime.IacRealtimeResult
	err = safeJSONUnmarshal(bytes.Bytes(), &results)
	assert.Nil(t, err, "Failed to unmarshal IAC results")

	// Verify response structure
	if len(results) > 0 {
		result := results[0]
		assert.NotEmpty(t, result.SimilarityID, "SimilarityID should not be empty")
		assert.NotEmpty(t, result.Title, "Title should not be empty")
		assert.NotEmpty(t, result.Severity, "Severity should not be empty")
		assert.NotEmpty(t, result.FilePath, "FilePath should not be empty")
		assert.Contains(t, []string{"Critical", "High", "Medium", "Low", "Info", "Unknown"},
			result.Severity, "Severity should be a valid value")
	}
}

func TestIacRealtimeScan_DockerFile_Success(t *testing.T) {
	configuration.LoadConfiguration()

	args := []string{
		"scan", "iac-realtime",
		flag(commonParams.SourcesFlag), "data/positive/Dockerfile",
		flag(commonParams.IacRealtimeEngineFlag), "docker",
	}

	err, bytes := executeCommand(t, args...)

	assert.Nil(t, err, "Scanning Dockerfile should not fail")

	var results []iacrealtime.IacRealtimeResult
	err = safeJSONUnmarshal(bytes.Bytes(), &results)
	assert.Nil(t, err, "Failed to unmarshal IAC results")

	// Verify response structure and content
	if len(results) > 0 {
		result := results[0]
		assert.NotEmpty(t, result.SimilarityID, "SimilarityID should not be empty")
		assert.NotEmpty(t, result.Title, "Title should not be empty")
		assert.NotEmpty(t, result.Severity, "Severity should not be empty")
		assert.Contains(t, result.FilePath, "Dockerfile", "FilePath should contain Dockerfile")

		// Verify locations exist
		if len(result.Locations) > 0 {
			location := result.Locations[0]
			assert.Greater(t, location.Line, 0, "Line number should be greater than 0")
		}
	}
}

func TestIacRealtimeScan_YamlConfigFile_Success(t *testing.T) {
	configuration.LoadConfiguration()

	args := []string{
		"scan", "iac-realtime",
		flag(commonParams.SourcesFlag), "data/config.yaml",
		flag(commonParams.IacRealtimeEngineFlag), "docker",
	}

	err, bytes := executeCommand(t, args...)

	// This might not find issues in a config file, but should not error
	assert.Nil(t, err, "Scanning YAML config file should not fail")

	var results []iacrealtime.IacRealtimeResult
	err = safeJSONUnmarshal(bytes.Bytes(), &results)
	assert.Nil(t, err, "Failed to unmarshal IAC results")

	// Results might be empty for a config file, but structure should be valid
	assert.NotNil(t, results, "Results should not be nil")
}

func TestIacRealtimeScan_EmptyDockerFile_Success(t *testing.T) {
	configuration.LoadConfiguration()

	args := []string{
		"scan", "iac-realtime",
		flag(commonParams.SourcesFlag), "data/empty/Dockerfile",
		flag(commonParams.IacRealtimeEngineFlag), "docker",
	}

	err, bytes := executeCommand(t, args...)

	assert.Nil(t, err, "Scanning empty Dockerfile should not fail")

	var results []iacrealtime.IacRealtimeResult
	err = safeJSONUnmarshal(bytes.Bytes(), &results)
	assert.Nil(t, err, "Failed to unmarshal IAC results")

	// Empty file might not have issues
	assert.NotNil(t, results, "Results should not be nil even for empty file")
}

func TestIacRealtimeScan_NonExistentFile_ShouldFail(t *testing.T) {
	configuration.LoadConfiguration()

	args := []string{
		"scan", "iac-realtime",
		flag(commonParams.SourcesFlag), "data/nonexistent.yaml",
		flag(commonParams.IacRealtimeEngineFlag), "docker",
	}

	err, _ := executeCommand(t, args...)

	assert.NotNil(t, err, "Scanning non-existent file should fail")
}

func TestIacRealtimeScan_UnsupportedFileExtension_ShouldFail(t *testing.T) {
	configuration.LoadConfiguration()

	// Create a temporary file with unsupported extension
	tempFile, err := os.CreateTemp("", "test-*.txt")
	assert.Nil(t, err, "Failed to create temp file")
	defer safeRemoveFile(tempFile.Name())

	_, err = tempFile.WriteString("some content")
	assert.Nil(t, err, "Failed to write to temp file")
	tempFile.Close()

	args := []string{
		"scan", "iac-realtime",
		flag(commonParams.SourcesFlag), tempFile.Name(),
		flag(commonParams.IacRealtimeEngineFlag), "docker",
	}

	err, _ = executeCommand(t, args...)

	assert.NotNil(t, err, "Scanning file with unsupported extension should fail")
}

func TestIacRealtimeScan_MissingSourceFlag_ShouldFail(t *testing.T) {
	configuration.LoadConfiguration()

	args := []string{
		"scan", "iac-realtime",
		flag(commonParams.IacRealtimeEngineFlag), "docker",
	}

	err, _ := executeCommand(t, args...)

	assert.NotNil(t, err, "Missing source flag should cause failure")
}

func TestIacRealtimeScan_WithIgnoredFilePath_Success(t *testing.T) {
	configuration.LoadConfiguration()

	// Create a temporary ignored file
	tempIgnoreFile, err := os.CreateTemp("", "ignore-*.json")
	assert.Nil(t, err, "Failed to create temp ignore file")
	defer safeRemoveFile(tempIgnoreFile.Name())

	// Write some ignore rules (empty for now)
	_, err = tempIgnoreFile.WriteString("[]")
	assert.Nil(t, err, "Failed to write to ignore file")
	tempIgnoreFile.Close()

	args := []string{
		"scan", "iac-realtime",
		flag(commonParams.SourcesFlag), "data/positive1.tf",
		flag(commonParams.IacRealtimeEngineFlag), "docker",
		flag(commonParams.IgnoredFilePathFlag), tempIgnoreFile.Name(),
	}

	err, bytes := executeCommand(t, args...)

	assert.Nil(t, err, "Scanning with ignored file path should not fail")

	var results []iacrealtime.IacRealtimeResult
	err = safeJSONUnmarshal(bytes.Bytes(), &results)
	assert.Nil(t, err, "Failed to unmarshal IAC results")
	assert.NotNil(t, results, "Results should not be nil")
}

func TestIacRealtimeScan_ResultsValidation_DetailedCheck(t *testing.T) {
	configuration.LoadConfiguration()

	args := []string{
		"scan", "iac-realtime",
		flag(commonParams.SourcesFlag), "data/positive1.tf",
		flag(commonParams.IacRealtimeEngineFlag), "docker",
	}

	err, bytes := executeCommand(t, args...)

	assert.Nil(t, err, "Scanning should not fail")

	var results []iacrealtime.IacRealtimeResult
	err = safeJSONUnmarshal(bytes.Bytes(), &results)
	assert.Nil(t, err, "Failed to unmarshal IAC results")

	// Detailed validation of result structure
	for _, result := range results {
		// Validate required fields
		assert.NotEmpty(t, result.SimilarityID, "Each result should have a SimilarityID")
		assert.NotEmpty(t, result.Title, "Each result should have a Title")
		assert.NotEmpty(t, result.Severity, "Each result should have a Severity")
		assert.NotEmpty(t, result.FilePath, "Each result should have a FilePath")

		// Validate severity values
		validSeverities := []string{"Critical", "High", "Medium", "Low", "Info", "Unknown"}
		assert.Contains(t, validSeverities, result.Severity,
			"Severity '%s' should be one of: %v", result.Severity, validSeverities)

		// Validate file path contains expected file
		assert.Contains(t, result.FilePath, "positive1.tf",
			"FilePath should contain the scanned file name")

		// Validate locations if present
		for _, location := range result.Locations {
			assert.GreaterOrEqual(t, location.Line, 0, "Line number should be 0 or positive")
			assert.GreaterOrEqual(t, location.StartIndex, 0, "StartIndex should be non-negative")
			assert.GreaterOrEqual(t, location.EndIndex, location.StartIndex,
				"EndIndex should be >= StartIndex")
		}
	}
}
