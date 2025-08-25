package secretsrealtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/2ms/v3/lib/reporting"
	"github.com/checkmarx/2ms/v3/lib/secrets"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewSecretsRealtimeService(t *testing.T) {
	jwtWrapper := &mock.JWTMockWrapper{}
	featureFlagWrapper := &mock.FeatureFlagsMockWrapper{}

	service := NewSecretsRealtimeService(jwtWrapper, featureFlagWrapper)

	assert.NotNil(t, service)
	assert.Equal(t, jwtWrapper, service.JwtWrapper)
	assert.Equal(t, featureFlagWrapper, service.FeatureFlagWrapper)
}

func TestRunSecretsRealtimeScan_EmptyFilePath_ReturnsError(t *testing.T) {
	service := &SecretsRealtimeService{
		JwtWrapper:         &mock.JWTMockWrapper{},
		FeatureFlagWrapper: &mock.FeatureFlagsMockWrapper{},
	}

	results, err := service.RunSecretsRealtimeScan("", "")

	assert.Nil(t, results)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "file path is required")
}

func TestRunSecretsRealtimeScan_FeatureFlagDisabled_ReturnsError(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: false}

	service := &SecretsRealtimeService{
		JwtWrapper:         &mock.JWTMockWrapper{},
		FeatureFlagWrapper: &mock.FeatureFlagsMockWrapper{},
	}

	results, err := service.RunSecretsRealtimeScan("test.txt", "")

	assert.Nil(t, results)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Realtime engine is not available")
}

func TestRunSecretsRealtimeScan_FileNotFound_ReturnsError(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}

	service := &SecretsRealtimeService{
		JwtWrapper:         &mock.JWTMockWrapper{},
		FeatureFlagWrapper: &mock.FeatureFlagsMockWrapper{},
	}

	results, err := service.RunSecretsRealtimeScan("nonexistent-file.txt", "")

	assert.Nil(t, results)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestRunSecretsRealtimeScan_WithIgnoreFile_FiltersResult(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "pat = \"ghp_1234567890abcdef123\""
	assert.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	ignoreFile := filepath.Join(tempDir, "ignored.json")
	ignored := []IgnoredSecret{
		{Title: "github-pat", SecretValue: "ghp_1234567890abcdef123"},
	}
	data, _ := json.Marshal(ignored)
	assert.NoError(t, os.WriteFile(ignoreFile, data, 0644))

	service := &SecretsRealtimeService{
		JwtWrapper:         &mock.JWTMockWrapper{},
		FeatureFlagWrapper: &mock.FeatureFlagsMockWrapper{},
	}

	results, err := service.RunSecretsRealtimeScan(testFile, ignoreFile)
	assert.NoError(t, err)
	assert.NotNil(t, results)

	for _, r := range results {
		assert.NotEqual(t, "github-pat", r.Title)
	}
}

func TestRunSecretsRealtimeScan_PatVulAndGenericVul_ReturnedOnlyPathVul(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "token = \"ghp_1234567890abcdef1234567890abcdef12345678\""
	assert.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	service := &SecretsRealtimeService{
		JwtWrapper:         &mock.JWTMockWrapper{},
		FeatureFlagWrapper: &mock.FeatureFlagsMockWrapper{},
	}

	results, err := service.RunSecretsRealtimeScan(testFile, "")
	assert.NoError(t, err)
	assert.NotNil(t, results)

	for _, r := range results {
		assert.Equal(t, "github-pat", r.Title)
	}
}

func TestRunSecretsRealtimeScan_ValidFile_Success(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}

	// Create a temporary file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test-secrets.txt")
	testContent := "aws_access_key_id = AKIAIOSFODNN7EXAMPLE\naws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	err := os.WriteFile(tempFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	service := &SecretsRealtimeService{
		JwtWrapper:         &mock.JWTMockWrapper{},
		FeatureFlagWrapper: &mock.FeatureFlagsMockWrapper{},
	}

	results, err := service.RunSecretsRealtimeScan(tempFile, "")

	assert.NoError(t, err)
	assert.NotNil(t, results)
}

func TestRunSecretsRealtimeScan_MultiLineResult_Success(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	value := "PRIVATE_KEY = \"\"\"\n-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA7v8wF+SECRETKEYEXAMPLE+QIDAQABAoIBAQC0\n-----END RSA PRIVATE KEY-----\n\"\"\""
	// Create a temporary file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test-secrets.txt")
	testContent := value
	err := os.WriteFile(tempFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	service := &SecretsRealtimeService{
		JwtWrapper:         &mock.JWTMockWrapper{},
		FeatureFlagWrapper: &mock.FeatureFlagsMockWrapper{},
	}

	results, err := service.RunSecretsRealtimeScan(tempFile, "")

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 1)
	assert.Len(t, results[0].Locations, 3)
	assert.NotEmpty(t, results[0].SecretValue)

}

func TestReadFile_ValidFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.txt")
	expectedContent := "test content"
	err := os.WriteFile(tempFile, []byte(expectedContent), 0644)
	assert.NoError(t, err)

	content, err := readFile(tempFile)

	assert.NoError(t, err)
	assert.Equal(t, expectedContent, content)
}

func TestReadFile_FileNotFound_ReturnsError(t *testing.T) {
	content, err := readFile("nonexistent-file.txt")

	assert.Empty(t, content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "readFile:")
}

func TestConvertToSecretsRealtimeResult_EmptyReport_ReturnsEmptySlice(t *testing.T) {
	report := &reporting.Report{
		Results: map[string][]*secrets.Secret{},
	}

	results := convertToSecretsRealtimeResult(report)

	assert.NotNil(t, results)
	assert.Len(t, results, 0)
}

func TestConvertToSecretsRealtimeResult_WithSecrets_ReturnsResults(t *testing.T) {
	secret1 := &secrets.Secret{
		RuleID:           "aws-access-key",
		RuleDescription:  "AWS Access Key",
		Source:           "test.txt",
		StartLine:        1,
		EndLine:          1,
		StartColumn:      0,
		EndColumn:        20,
		ValidationStatus: validSecret,
	}

	secret2 := &secrets.Secret{
		RuleID:           "github-token",
		RuleDescription:  "GitHub Token",
		Source:           "test.txt",
		StartLine:        2,
		EndLine:          2,
		StartColumn:      0,
		EndColumn:        30,
		ValidationStatus: invalidSecret,
	}

	report := &reporting.Report{
		Results: map[string][]*secrets.Secret{
			"test.txt": {secret1, secret2},
		},
	}

	results := convertToSecretsRealtimeResult(report)

	assert.NotNil(t, results)
	assert.Len(t, results, 2)

	// Check first result
	assert.Equal(t, "aws-access-key", results[0].Title)
	assert.Equal(t, "AWS Access Key", results[0].Description)
	assert.Equal(t, "test.txt", results[0].FilePath)
	assert.Equal(t, criticalSeverity, results[0].Severity)
	assert.Len(t, results[0].Locations, 1)
	assert.Equal(t, 1, results[0].Locations[0].Line)
	assert.Equal(t, 0, results[0].Locations[0].StartIndex)
	assert.Equal(t, 20, results[0].Locations[0].EndIndex)

	// Check second result
	assert.Equal(t, "github-token", results[1].Title)
	assert.Equal(t, "GitHub Token", results[1].Description)
	assert.Equal(t, "test.txt", results[1].FilePath)
	assert.Equal(t, mediumSeverity, results[1].Severity)
	assert.Len(t, results[1].Locations, 1)
	assert.Equal(t, 2, results[1].Locations[0].Line)
	assert.Equal(t, 0, results[1].Locations[0].StartIndex)
	assert.Equal(t, 30, results[1].Locations[0].EndIndex)
}

func TestConvertSecretToResult_SingleLineSecret_Success(t *testing.T) {
	secret := &secrets.Secret{
		RuleID:           "test-rule",
		RuleDescription:  "Test Rule Description",
		Source:           "test-file.txt",
		StartLine:        5,
		EndLine:          5,
		StartColumn:      10,
		EndColumn:        25,
		ValidationStatus: validSecret,
	}

	result := convertSecretToResult(secret)

	assert.Equal(t, "test-rule", result.Title)
	assert.Equal(t, "Test Rule Description", result.Description)
	assert.Equal(t, "test-file.txt", result.FilePath)
	assert.Equal(t, criticalSeverity, result.Severity)
	assert.Len(t, result.Locations, 1)
	assert.Equal(t, 5, result.Locations[0].Line)
	assert.Equal(t, 10, result.Locations[0].StartIndex)
	assert.Equal(t, 25, result.Locations[0].EndIndex)
}

func TestConvertSecretToResult_MultiLineSecret_Success(t *testing.T) {
	secret := &secrets.Secret{
		RuleID:           "multiline-rule",
		RuleDescription:  "Multi-line Rule",
		Source:           "test-file.txt",
		StartLine:        3,
		EndLine:          5,
		StartColumn:      0,
		EndColumn:        10,
		ValidationStatus: unknownSecret,
	}

	result := convertSecretToResult(secret)

	assert.Equal(t, "multiline-rule", result.Title)
	assert.Equal(t, "Multi-line Rule", result.Description)
	assert.Equal(t, "test-file.txt", result.FilePath)
	assert.Equal(t, highSeverity, result.Severity)
	assert.Len(t, result.Locations, 3) // Lines 3, 4, 5

	expectedLines := []int{3, 4, 5}
	for i, location := range result.Locations {
		assert.Equal(t, expectedLines[i], location.Line)
		assert.Equal(t, 0, location.StartIndex)
		assert.Equal(t, 10, location.EndIndex)
	}
}

func TestGetSeverity_ValidSecret_ReturnsCritical(t *testing.T) {
	secret := &secrets.Secret{ValidationStatus: validSecret}

	severity := getSeverity(secret)

	assert.Equal(t, criticalSeverity, severity)
}

func TestGetSeverity_UnknownSecret_ReturnsHigh(t *testing.T) {
	secret := &secrets.Secret{ValidationStatus: unknownSecret}

	severity := getSeverity(secret)

	assert.Equal(t, highSeverity, severity)
}

func TestGetSeverity_InvalidSecret_ReturnsMedium(t *testing.T) {
	secret := &secrets.Secret{ValidationStatus: invalidSecret}

	severity := getSeverity(secret)

	assert.Equal(t, mediumSeverity, severity)
}

func TestGetSeverity_UnknownStatus_ReturnsHigh(t *testing.T) {
	secret := &secrets.Secret{ValidationStatus: "SomeUnknownStatus"}

	severity := getSeverity(secret)

	assert.Equal(t, highSeverity, severity)
}

func TestGetSeverity_EmptyStatus_ReturnsHigh(t *testing.T) {
	secret := &secrets.Secret{ValidationStatus: ""}

	severity := getSeverity(secret)

	assert.Equal(t, highSeverity, severity)
}
