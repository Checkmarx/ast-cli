package commands

import (
	"reflect"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
)

func Test_ExecuteVorpalScan(t *testing.T) {
	type args struct {
		fileSourceFlag      string
		vorpalUpdateVersion bool
	}
	tests := []struct {
		name       string
		args       args
		want       *grpcs.ScanResult
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "Test with empty fileSource flag should not return error",
			args: args{
				fileSourceFlag:      "",
				vorpalUpdateVersion: true,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Test path to file without extension",
			args: args{
				fileSourceFlag:      "data/python-vul-file",
				vorpalUpdateVersion: false,
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: errorConstants.FileExtensionIsRequired,
		},
		{
			name: "Test with valid flags. vorpalUpdateVersion set to true",
			args: args{
				fileSourceFlag:      "data/python-vul-file.py",
				vorpalUpdateVersion: true,
			},
			//TODO: update mocks when there's a real engine
			want:    ReturnSuccessfulResponseMock(),
			wantErr: false,
		},
		{
			name: "Test with valid flags. vorpalUpdateVersion set to false",
			args: args{
				fileSourceFlag:      "data/python-vul-file.py",
				vorpalUpdateVersion: false,
			},
			//TODO: update mocks when there's a real engine
			want:    ReturnFailureResponseMock(),
			wantErr: false,
		},
		{
			name: "Test with valid flags and no vulnerabilities in file",
			args: args{
				fileSourceFlag:      "data/csharp-no-vul.cs",
				vorpalUpdateVersion: false,
			},
			//TODO: update mocks when there's a real engine
			want:    ReturnFailureResponseMock(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(ttt.name, func(t *testing.T) {
			vorpalParams := services.VorpalScanParams{
				FilePath:            ttt.args.fileSourceFlag,
				VorpalUpdateVersion: ttt.args.vorpalUpdateVersion,
				IsDefaultAgent:      true,
				JwtWrapper:          &mock.JWTMockWrapper{},
				FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
			}
			got, err := ExecuteVorpalScan(vorpalParams)
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
		want       *grpcs.ScanResult
		wantErrMsg string
	}{
		{
			name:       "Test with empty fileSourceFlag",
			sourceFlag: "",
			engineFlag: true,
			wantErr:    false,
			want:       nil,
		},
		{
			name:       "Test with file without extension",
			sourceFlag: "data/python-vul-file",
			engineFlag: true,
			wantErr:    true,
			wantErrMsg: errorConstants.FileExtensionIsRequired,
		},
		{
			name:       "Test with valid fileSource Flag and vorpalUpdateVersion flag set false ",
			sourceFlag: "data/python-vul-file.py",
			engineFlag: false,
			want:       nil,
			wantErr:    false,
		},
		{
			name:       "Test with valid fileSource Flag and vorpalUpdateVersion flag set true ",
			sourceFlag: "data/python-vul-file.py",
			engineFlag: true,
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
			runFunc := runScanVorpalCommand(&mock.JWTMockWrapper{}, &mock.FeatureFlagsMockWrapper{})
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
