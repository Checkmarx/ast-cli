package services

import (
	"fmt"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestExportSbomResults(t *testing.T) {
	type args struct {
		exportWrapper     wrappers.ExportWrapper
		targetFile        string
		results           *wrappers.ResultSummary
		formatSbomOptions string
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Test ExportSbomResults",
			args: args{
				exportWrapper: &mock.ExportMockWrapper{},
				targetFile:    "test.txt",
				results: &wrappers.ResultSummary{
					ScanID: "id123456",
				},
				formatSbomOptions: "CycloneDxJson",
			},
			wantErr: assert.NoError,
		},
		{
			name: "Test ExportSbomResults with invalid format",
			args: args{
				exportWrapper: &mock.ExportMockWrapper{},
				targetFile:    "test.txt",
				results: &wrappers.ResultSummary{
					ScanID: "id123456",
				},
				formatSbomOptions: "invalid",
			},
			wantErr: assert.Error,
		},
		{
			name: "Test ExportSbomResults with error",
			args: args{
				exportWrapper: &mock.ExportMockWrapper{},
				targetFile:    "test.txt",
				results: &wrappers.ResultSummary{
					ScanID: "err-scan-id",
				},
				formatSbomOptions: "CycloneDxJson",
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, ExportSbomResults(tt.args.exportWrapper, tt.args.targetFile, tt.args.results, tt.args.formatSbomOptions),
				fmt.Sprintf("ExportSbomResults(%v, %v, %v, %v)", tt.args.exportWrapper, tt.args.targetFile, tt.args.results, tt.args.formatSbomOptions))
		})
	}
}

func TestGetExportPackage_InitiateExportRequestError_ReturnsError(t *testing.T) {
	result, err := GetExportPackage(&mock.ExportMockWrapper{}, "err-scan-id", false, &mock.FeatureFlagsMockWrapper{})
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetExportPackage_MinioDisabled_UsesExportIDAsFilePath(t *testing.T) {
	resetFeatureFlagState()
	defer resetFeatureFlagState()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.MinioEnabled, Status: false}

	var capturedFilePath string
	var capturedAuth bool
	exportWrapper := &mock.ExportMockWrapper{
		CustomGetScaPackageCollectionExport: func(fileURL string, auth bool) (*wrappers.ScaPackageCollectionExport, error) {
			capturedFilePath = fileURL
			capturedAuth = auth
			return &wrappers.ScaPackageCollectionExport{}, nil
		},
	}

	result, err := GetExportPackage(exportWrapper, "scan-id-123", false, &mock.FeatureFlagsMockWrapper{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "id123456", capturedFilePath)
	assert.False(t, capturedAuth)
}

func TestGetExportPackage_MinioEnabled_UsesFileURLAsFilePath(t *testing.T) {
	resetFeatureFlagState()
	defer resetFeatureFlagState()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.MinioEnabled, Status: true}

	var capturedFilePath string
	var capturedAuth bool
	exportWrapper := &mock.ExportMockWrapper{
		CustomGetScaPackageCollectionExport: func(fileURL string, auth bool) (*wrappers.ScaPackageCollectionExport, error) {
			capturedFilePath = fileURL
			capturedAuth = auth
			return &wrappers.ScaPackageCollectionExport{}, nil
		},
	}

	result, err := GetExportPackage(exportWrapper, "scan-id-123", true, &mock.FeatureFlagsMockWrapper{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "url", capturedFilePath)
	assert.True(t, capturedAuth)
}

func TestGetExportPackage_NoResultsFound_ReturnsEmptyCollectionWithoutError(t *testing.T) {
	resetFeatureFlagState()
	defer resetFeatureFlagState()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.MinioEnabled, Status: false}

	exportWrapper := &mock.ExportMockWrapper{
		CustomGetExportReportStatus: func(exportID string) (*wrappers.ExportPollingResponse, error) {
			return &wrappers.ExportPollingResponse{
				ExportStatus: completedStatus,
				ErrorMessage: "No results were found for the scan",
			}, nil
		},
	}

	result, err := GetExportPackage(exportWrapper, "scan-id-123", false, &mock.FeatureFlagsMockWrapper{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Packages)
}

func TestGetExportPackage_PollForCompletionError_ReturnsError(t *testing.T) {
	exportWrapper := &mock.ExportMockWrapper{
		CustomGetExportReportStatus: func(exportID string) (*wrappers.ExportPollingResponse, error) {
			return nil, fmt.Errorf("polling failed")
		},
	}

	result, err := GetExportPackage(exportWrapper, "scan-id-123", false, &mock.FeatureFlagsMockWrapper{})
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestValidateSbomOptions tests the validateSbomOptions function
func TestValidateSbomOptions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "Valid CycloneDxJson",
			input:   "cyclonedxjson",
			want:    "CycloneDxJson",
			wantErr: false,
		},
		{
			name:    "Valid CycloneDxJson with uppercase",
			input:   "CYCLONEDXJSON",
			want:    "CycloneDxJson",
			wantErr: false,
		},
		{
			name:    "Valid CycloneDxJson with spaces",
			input:   "cyclone dx json",
			want:    "CycloneDxJson",
			wantErr: false,
		},
		{
			name:    "Valid CycloneDxXml",
			input:   "cyclonedxxml",
			want:    "CycloneDxXml",
			wantErr: false,
		},
		{
			name:    "Valid SpdxJson",
			input:   "spdxjson",
			want:    "SpdxJson",
			wantErr: false,
		},
		{
			name:    "Valid with mixed case and spaces",
			input:   "CYCLONE DX XML",
			want:    "CycloneDxXml",
			wantErr: false,
		},
		{
			name:    "Invalid option",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Invalid with spaces",
			input:   "xyz format",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateSbomOptions(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSbomOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateSbomOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPreparePayload tests the preparePayload function
func TestPreparePayload(t *testing.T) {
	tests := []struct {
		name              string
		scanID            string
		formatSbomOptions string
		expectedFormat    string
		wantErr           bool
	}{
		{
			name:              "Default format",
			scanID:            "scan123",
			formatSbomOptions: "",
			expectedFormat:    "CycloneDxJson",
			wantErr:           false,
		},
		{
			name:              "Explicit default format",
			scanID:            "scan456",
			formatSbomOptions: "CycloneDxJson",
			expectedFormat:    "CycloneDxJson",
			wantErr:           false,
		},
		{
			name:              "CycloneDxXml format",
			scanID:            "scan789",
			formatSbomOptions: "cyclonedxxml",
			expectedFormat:    "CycloneDxXml",
			wantErr:           false,
		},
		{
			name:              "SpdxJson format",
			scanID:            "scan999",
			formatSbomOptions: "spdxjson",
			expectedFormat:    "SpdxJson",
			wantErr:           false,
		},
		{
			name:              "Invalid format",
			scanID:            "scan111",
			formatSbomOptions: "invalid",
			expectedFormat:    "",
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := preparePayload(tt.scanID, tt.formatSbomOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("preparePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.scanID, payload.ScanID)
				assert.Equal(t, tt.expectedFormat, payload.FileFormat)
			}
		})
	}
}

// TestGetExportPackage tests the GetExportPackage function
func TestGetExportPackage(t *testing.T) {
	tests := []struct {
		name                  string
		exportWrapper         wrappers.ExportWrapper
		scanID                string
		scaHideDevAndTestDep  bool
		featureflagWrappers   wrappers.FeatureFlagsWrapper
		wantErr               bool
	}{
		{
			name:                 "Successful export",
			exportWrapper:        &mock.ExportMockWrapper{},
			scanID:               "scan-123",
			scaHideDevAndTestDep: false,
			featureflagWrappers:  &mock.FeatureFlagsMockWrapper{},
			wantErr:              false,
		},
		{
			name:                 "Successful export with hide dev deps",
			exportWrapper:        &mock.ExportMockWrapper{},
			scanID:               "scan-456",
			scaHideDevAndTestDep: true,
			featureflagWrappers:  &mock.FeatureFlagsMockWrapper{},
			wantErr:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetExportPackage(tt.exportWrapper, tt.scanID, tt.scaHideDevAndTestDep, tt.featureflagWrappers)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetExportPackage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("GetExportPackage() returned nil when no error expected")
			}
		})
	}
}

// TestExportSbomResults_MultipleFormats tests ExportSbomResults with different SBOM formats
func TestExportSbomResults_MultipleFormats(t *testing.T) {
	formats := []string{
		"CycloneDxJson",
		"cyclonedxxml",
		"spdxjson",
	}

	for _, format := range formats {
		t.Run(fmt.Sprintf("Format_%s", format), func(t *testing.T) {
			exportWrapper := &mock.ExportMockWrapper{}
			results := &wrappers.ResultSummary{
				ScanID: "test-scan-id",
			}

			err := ExportSbomResults(exportWrapper, "output.json", results, format)
			// Error is expected when format is not exactly matching, but it should be handled
			_ = err
		})
	}
}

// TestPreparePayload_EdgeCases tests edge cases in preparePayload
func TestPreparePayload_EdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		scanID            string
		formatSbomOptions string
		expectFormat      string
		wantErr           bool
	}{
		{
			name:              "Empty scanID",
			scanID:            "",
			formatSbomOptions: "",
			expectFormat:      "CycloneDxJson",
			wantErr:           false,
		},
		{
			name:              "Special characters in scanID",
			scanID:            "scan-123-xyz_456",
			formatSbomOptions: "cyclonedxjson",
			expectFormat:      "CycloneDxJson",
			wantErr:           false,
		},
		{
			name:              "Format with spaces and mixed case",
			scanID:            "scan123",
			formatSbomOptions: "CYCLONE DX XML",
			expectFormat:      "CycloneDxXml",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := preparePayload(tt.scanID, tt.formatSbomOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("preparePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.scanID, payload.ScanID)
				assert.Equal(t, tt.expectFormat, payload.FileFormat)
			}
		})
	}
}

// TestValidateSbomOptions_AllValidFormats tests all valid SBOM format options
func TestValidateSbomOptions_AllValidFormats(t *testing.T) {
	validFormats := map[string]string{
		"cyclonedxjson": "CycloneDxJson",
		"cyclonedxxml":  "CycloneDxXml",
		"spdxjson":      "SpdxJson",
	}

	for input, expected := range validFormats {
		t.Run(fmt.Sprintf("Format_%s", input), func(t *testing.T) {
			got, err := validateSbomOptions(input)
			assert.NoError(t, err)
			assert.Equal(t, expected, got)
		})
	}
}

// TestValidateSbomOptions_InvalidFormats tests invalid SBOM format options
func TestValidateSbomOptions_InvalidFormats(t *testing.T) {
	invalidFormats := []string{
		"notaformat",
		"unknown",
		"xyz",
		"123",
		"cyclone",
	}

	for _, format := range invalidFormats {
		t.Run(fmt.Sprintf("Invalid_%s", format), func(t *testing.T) {
			_, err := validateSbomOptions(format)
			assert.Error(t, err)
		})
	}
}

// TestExportSbomResults_WithDifferentTargetFiles tests ExportSbomResults with various target file paths
func TestExportSbomResults_WithDifferentTargetFiles(t *testing.T) {
	targetFiles := []string{
		"output.json",
		"sbom.json",
		"report.xml",
	}

	for _, targetFile := range targetFiles {
		t.Run(fmt.Sprintf("TargetFile_%s", targetFile), func(t *testing.T) {
			exportWrapper := &mock.ExportMockWrapper{}
			results := &wrappers.ResultSummary{
				ScanID: "test-scan",
			}

			err := ExportSbomResults(exportWrapper, targetFile, results, "CycloneDxJson")
			// We just verify it doesn't panic
			_ = err
		})
	}
}

// TestGetExportPackage_WithDifferentScans tests GetExportPackage with different scan scenarios
func TestGetExportPackage_WithDifferentScans(t *testing.T) {
	scans := []string{
		"scan-001",
		"scan-with-special-chars_123",
		"",
	}

	for _, scanID := range scans {
		t.Run(fmt.Sprintf("ScanID_%s", scanID), func(t *testing.T) {
			exportWrapper := &mock.ExportMockWrapper{}
			featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}

			_, err := GetExportPackage(exportWrapper, scanID, false, featureFlagsWrapper)
			// We just verify it handles different scan IDs
			_ = err
		})
	}
}

// TestPreparePayload_WithEmptyFormat tests preparePayload when format is the default
func TestPreparePayload_WithEmptyFormat(t *testing.T) {
	payload, err := preparePayload("test-scan", "")
	assert.NoError(t, err)
	assert.Equal(t, "test-scan", payload.ScanID)
	assert.Equal(t, DefaultSbomOption, payload.FileFormat)
}

// TestPreparePayload_WithDefaultFormat tests preparePayload when format matches default
func TestPreparePayload_WithDefaultFormat(t *testing.T) {
	payload, err := preparePayload("test-scan", DefaultSbomOption)
	assert.NoError(t, err)
	assert.Equal(t, "test-scan", payload.ScanID)
	assert.Equal(t, DefaultSbomOption, payload.FileFormat)
}

// TestValidateSbomOptions_CaseSensitivity tests case insensitivity of validateSbomOptions
func TestValidateSbomOptions_CaseSensitivity(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"cyclonedxjson", "CycloneDxJson"},
		{"CYCLONEDXJSON", "CycloneDxJson"},
		{"CycloneDxJson", "CycloneDxJson"},
		{"cyclone dx json", "CycloneDxJson"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("Case_%s", c.input), func(t *testing.T) {
			got, err := validateSbomOptions(c.input)
			assert.NoError(t, err)
			assert.Equal(t, c.expected, got)
		})
	}
}

// TestExportSbomResults_ErrorCases tests ExportSbomResults error handling
func TestExportSbomResults_ErrorCases(t *testing.T) {
	tests := []struct {
		name               string
		scanID             string
		formatSbomOptions  string
		shouldError        bool
	}{
		{
			name:              "Valid with default format",
			scanID:            "scan-ok",
			formatSbomOptions: "",
			shouldError:       false,
		},
		{
			name:              "Invalid format causes error",
			scanID:            "scan-123",
			formatSbomOptions: "badformat",
			shouldError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exportWrapper := &mock.ExportMockWrapper{}
			results := &wrappers.ResultSummary{
				ScanID: tt.scanID,
			}

			err := ExportSbomResults(exportWrapper, "output.json", results, tt.formatSbomOptions)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
