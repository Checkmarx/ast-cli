package ASCA

import (
	"reflect"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
)

func Test_ExecuteASCAScan(t *testing.T) {
	type args struct {
		fileSourceFlag    string
		ASCAUpdateVersion bool
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
				fileSourceFlag:    "",
				ASCAUpdateVersion: true,
			},
			want: &grpcs.ScanResult{
				Message: services.FilePathNotProvided,
			},
			wantErr: false,
		},
		{
			name: "Test with valid flags. ASCAUpdateVersion set to true",
			args: args{
				fileSourceFlag:    "../data/python-vul-file.py",
				ASCAUpdateVersion: true,
			},
			want:    mock.ReturnSuccessfulResponseMock(),
			wantErr: false,
		},
		{
			name: "Test with valid flags. ASCAUpdateVersion set to false",
			args: args{
				fileSourceFlag:    "../data/python-vul-file.py",
				ASCAUpdateVersion: false,
			},
			want:    mock.ReturnSuccessfulResponseMock(),
			wantErr: false,
		},
		{
			name: "Test with valid flags. ASCA scan failed",
			args: args{
				fileSourceFlag:    "../data/csharp-no-vul.cs",
				ASCAUpdateVersion: false,
			},
			want:    mock.ReturnFailureResponseMock(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(ttt.name, func(t *testing.T) {
			ASCAParams := services.ASCAScanParams{
				FilePath:          ttt.args.fileSourceFlag,
				ASCAUpdateVersion: ttt.args.ASCAUpdateVersion,
				IsDefaultAgent:    true,
			}
			wrapperParams := services.ASCAWrappersParam{
				JwtWrapper:          &mock.JWTMockWrapper{},
				FeatureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
				ASCAWrapper:         &mock.ASCAMockWrapper{},
			}
			got, err := services.CreateASCAScanRequest(ASCAParams, wrapperParams)
			if (err != nil) != ttt.wantErr {
				t.Errorf("executeASCAScan() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if ttt.wantErr && err.Error() != ttt.wantErrMsg {
				t.Errorf("executeASCAScan() error message = %v, wantErrMsg %v", err.Error(), ttt.wantErrMsg)
			}
			if !reflect.DeepEqual(got, ttt.want) {
				t.Errorf("executeASCAScan() got = %v, want %v", got, ttt.want)
			}
		})
	}
}

func Test_runScanASCACommand(t *testing.T) {
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
			name:       "Test with valid fileSource Flag and ASCAUpdateVersion flag set false ",
			sourceFlag: "data/python-vul-file.py",
			engineFlag: false,
			want:       nil,
			wantErr:    false,
		},
		{
			name:       "Test with valid fileSource Flag and ASCAUpdateVersion flag set true ",
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
			cmd.Flags().Bool(commonParams.ASCALatestVersion, ttt.engineFlag, "")
			cmd.Flags().String(commonParams.FormatFlag, printer.FormatJSON, "")
			runFunc := RunScanASCACommand(&mock.JWTMockWrapper{}, &mock.FeatureFlagsMockWrapper{})
			err := runFunc(cmd, []string{})
			if (err != nil) != ttt.wantErr {
				t.Errorf("RunScanASCACommand() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if ttt.wantErr && err.Error() != ttt.wantErrMsg {
				t.Errorf("RunScanASCACommand() error message = %v, wantErrMsg %v", err.Error(), ttt.wantErrMsg)
			}
		})
	}
}
