package commands

import (
	"reflect"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/cobra"
)

func Test_ExecuteVorpalScan(t *testing.T) {
	type args struct {
		sourceFlag          string
		vorpalUpdateVersion bool
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
				vorpalUpdateVersion: true,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Test with file without extension",
			args: args{
				sourceFlag:          "source",
				vorpalUpdateVersion: false,
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: errorConstants.FileExtensionIsRequired,
		},
		{
			name: "Test with correct flags",
			args: args{
				sourceFlag:          "source.cs",
				vorpalUpdateVersion: true,
			},
			want:    ReturnSuccessfulResponseMock(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(ttt.name, func(t *testing.T) {
			got, err := ExecuteVorpalScan(ttt.args.sourceFlag, ttt.args.vorpalUpdateVersion)
			if (err != nil) != ttt.wantErr {
				t.Errorf("executeVorpalScan() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if ttt.wantErr && err.Error() != ttt.wantErrMsg {
				t.Errorf("executeVorpalScan() error message = %v, wantErrMsg %v", err.Error(), ttt.wantErrMsg)
			}
			if !reflect.DeepEqual(got, ttt.want) {
				t.Errorf("executeVorpalScan() got = %v, want %v", got, ttt.want)
			}
		})
	}
}

func Test_runScanVorpalCommand(t *testing.T) {
	tests := []struct {
		name       string
		sourceFlag string
		engineFlag bool
		wantErr    bool
		want       *ScanResult
		wantErrMsg string
	}{
		{
			name:       "Test with empty sourceFlag",
			sourceFlag: "",
			engineFlag: true,
			wantErr:    false,
			want:       nil,
		},
		{
			name:       "Test with file without extension",
			sourceFlag: "source",
			engineFlag: true,
			wantErr:    true,
			wantErrMsg: errorConstants.FileExtensionIsRequired,
		},
		{
			name:       "Test with valid sourceFlag and engineFlag",
			sourceFlag: "source.cs",
			engineFlag: false,
			want:       nil,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(ttt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String(commonParams.SourcesFlag, ttt.sourceFlag, "")
			cmd.Flags().Bool(commonParams.VorpalLatestVersion, ttt.engineFlag, "")
			cmd.Flags().String(commonParams.FormatFlag, printer.FormatJSON, "")
			runFunc := runScanVorpalCommand()
			err := runFunc(cmd, []string{})
			if (err != nil) != ttt.wantErr {
				t.Errorf("runScanVorpalCommand() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if ttt.wantErr && err.Error() != ttt.wantErrMsg {
				t.Errorf("runScanVorpalCommand() error message = %v, wantErrMsg %v", err.Error(), ttt.wantErrMsg)
			}
		})
	}
}
