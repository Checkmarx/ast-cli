package commands

import (
	"reflect"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/cobra"
)

func Test_executeLightweightScan(t *testing.T) {
	type args struct {
		sourceFlag          string
		engineUpdateVersion string
	}
	tests := []struct {
		name       string
		args       args
		want       *ScanResult
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "Test with empty sourceFlag",
			args: args{
				sourceFlag:          "",
				engineUpdateVersion: "1.0.0",
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: errorConstants.SourceCodeIsRequired,
		},
		{
			name: "Test with empty engineUpdateVersion",
			args: args{
				sourceFlag:          "source.cs",
				engineUpdateVersion: "",
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: errorConstants.EngineVersionIsRequired,
		},
		{
			name: "Test with file without extension",
			args: args{
				sourceFlag:          "source",
				engineUpdateVersion: "1.0.0",
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: errorConstants.FileExtensionIsRequired,
		},
		{
			name: "Test with empty engineUpdateVersion",
			args: args{
				sourceFlag:          "source.cs",
				engineUpdateVersion: "1.0.0",
			},
			want:    returnSuccessfulResponseMock(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExecuteLightweightScan(tt.args.sourceFlag, tt.args.engineUpdateVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeLightweightScan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.wantErrMsg {
				t.Errorf("executeLightweightScan() error message = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeLightweightScan() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_runScanLightweightCommand(t *testing.T) {
	tests := []struct {
		name       string
		sourceFlag string
		engineFlag string
		wantErr    bool
		want       *ScanResult
		wantErrMsg string
	}{
		{
			name:       "Test with empty sourceFlag",
			sourceFlag: "",
			engineFlag: "1.0.0",
			wantErr:    true,
			wantErrMsg: errorConstants.SourceCodeIsRequired,
		},
		{
			name:       "Test with empty engineFlag",
			sourceFlag: "source.cs",
			engineFlag: "",
			wantErr:    true,
			wantErrMsg: errorConstants.EngineVersionIsRequired,
		},
		{
			name:       "Test with file without extension",
			sourceFlag: "source",
			engineFlag: "1.0.0",
			wantErr:    true,
			wantErrMsg: errorConstants.FileExtensionIsRequired,
		},
		{
			name:       "Test with valid sourceFlag and engineFlag",
			sourceFlag: "source.cs",
			engineFlag: "1.0.0",
			want:       nil,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String(commonParams.SourcesFlag, tt.sourceFlag, "")
			cmd.Flags().String(commonParams.LightweightUpdateVersion, tt.engineFlag, "")
			cmd.Flags().String(commonParams.FormatFlag, printer.FormatTable, "")
			runFunc := runScanLightweightCommand()
			err := runFunc(cmd, []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("runScanLightweightCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.wantErrMsg {
				t.Errorf("runScanLightweightCommand() error message = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
			}
		})
	}
}
