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
