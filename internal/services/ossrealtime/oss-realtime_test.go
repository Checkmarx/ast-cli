package ossrealtime

import (
	"reflect"
	"testing"

	"github.com/Checkmarx/manifest-parser/pkg/models"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

func TestRunOssRealtimeScan(t *testing.T) {
	type args struct {
		realtimeScannerWrapperParams *RealtimeScannerWrapperParams
		filePath                     string
	}
	tests := []struct {
		name    string
		args    args
		want    *wrappers.OssPackageResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RunOssRealtimeScan(tt.args.realtimeScannerWrapperParams, tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ensureLicense(t *testing.T) {
	type args struct {
		realtimeScannerWrapperParams *RealtimeScannerWrapperParams
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ensureLicense(tt.args.realtimeScannerWrapperParams); (err != nil) != tt.wantErr {
				t.Errorf("ensureLicense() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_parseManifest(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		want    []models.Package
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseManifest(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseManifest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseManifest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pkgToRequest(t *testing.T) {
	type args struct {
		pkg models.Package
	}
	tests := []struct {
		name string
		args args
		want wrappers.OssPackageRequest
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pkgToRequest(tt.args.pkg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pkgToRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_prepareScan(t *testing.T) {
	type args struct {
		pkgs []models.Package
	}
	tests := []struct {
		name  string
		args  args
		want  wrappers.OssPackageResponse
		want1 []wrappers.OssPackageRequest
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := prepareScan(tt.args.pkgs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prepareScan() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("prepareScan() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_scanAndCache(t *testing.T) {
	type args struct {
		realtimeScannerWrapperParams *RealtimeScannerWrapperParams
		requestPackages              []wrappers.OssPackageRequest
		resp                         *wrappers.OssPackageResponse
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := scanAndCache(tt.args.realtimeScannerWrapperParams, tt.args.requestPackages, tt.args.resp); (err != nil) != tt.wantErr {
				t.Errorf("scanAndCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
