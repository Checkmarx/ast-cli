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
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, ExportSbomResults(tt.args.exportWrapper, tt.args.targetFile, tt.args.results, tt.args.formatSbomOptions), fmt.Sprintf("ExportSbomResults(%v, %v, %v, %v)", tt.args.exportWrapper, tt.args.targetFile, tt.args.results, tt.args.formatSbomOptions))
		})
	}
}
