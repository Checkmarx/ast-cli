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
		engineUpdateVersion bool
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
				engineUpdateVersion: true,
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: errorConstants.FileSourceFlagIsRequired,
		},
		{
			name: "Test with file without extension",
			args: args{
				sourceFlag:          "source",
				engineUpdateVersion: false,
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: errorConstants.FileExtensionIsRequired,
		},
		{
			name: "Test with correct flags",
			args: args{
				sourceFlag:          "source.cs",
				engineUpdateVersion: true,
			},
			want:    ReturnSuccessfulResponseMock(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(ttt.name, func(t *testing.T) {
			got, err := ExecuteLightweightScan(ttt.args.sourceFlag, ttt.args.engineUpdateVersion)
			if (err != nil) != ttt.wantErr {
				t.Errorf("executeLightweightScan() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if ttt.wantErr && err.Error() != ttt.wantErrMsg {
				t.Errorf("executeLightweightScan() error message = %v, wantErrMsg %v", err.Error(), ttt.wantErrMsg)
			}
			if !reflect.DeepEqual(got, ttt.want) {
				t.Errorf("executeLightweightScan() got = %v, want %v", got, ttt.want)
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
			engineFlag: "true",
			wantErr:    true,
			wantErrMsg: errorConstants.FileSourceFlagIsRequired,
		},
		{
			name:       "Test with file without extension",
			sourceFlag: "source",
			engineFlag: "true",
			wantErr:    true,
			wantErrMsg: errorConstants.FileExtensionIsRequired,
		},
		{
			name:       "Test with valid sourceFlag and engineFlag",
			sourceFlag: "source.cs",
			engineFlag: "false",
			want:       nil,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(ttt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String(commonParams.SourcesFlag, ttt.sourceFlag, "")
			cmd.Flags().String(commonParams.LightweightUpdateVersion, ttt.engineFlag, "")
			cmd.Flags().String(commonParams.FormatFlag, printer.FormatJSON, "")
			runFunc := runScanLightweightCommand()
			err := runFunc(cmd, []string{})
			if (err != nil) != ttt.wantErr {
				t.Errorf("runScanLightweightCommand() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if ttt.wantErr && err.Error() != ttt.wantErrMsg {
				t.Errorf("runScanLightweightCommand() error message = %v, wantErrMsg %v", err.Error(), ttt.wantErrMsg)
			}
		})
	}
}
