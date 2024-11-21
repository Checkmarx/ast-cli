package configuration

import (
	"reflect"
	"strings"
	"testing"

	asserts "github.com/stretchr/testify/assert"
)

func TestGetConfigFilePath(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			name:    "Check if the config file path is correct",
			want:    ".checkmarx/checkmarxcli.yaml",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			got, err := GetConfigFilePath()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigFilePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			asserts.True(t, strings.HasSuffix(got, tt.want))
		})
	}
}

func TestWriteSingleConfigKey(t *testing.T) {
	type args struct {
		configFilePath string
		key            string
		value          int
	}
	configFilePath, _ := GetConfigFilePath()
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Write single config key to existing file",
			args: args{
				configFilePath: configFilePath,
				key:            "cx_asca_port",
				value:          0,
			},
			wantErr: false,
		},
		{
			name: "Write single config key to non-existing file",
			args: args{
				configFilePath: "non-existing-file",
				key:            "cx_asca_port",
				value:          0,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := WriteSingleConfigKey(tt.args.configFilePath, tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("WriteSingleConfigKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_loadConfig(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadConfig(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_obfuscateString(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := obfuscateString(tt.args.str); got != tt.want {
				t.Errorf("obfuscateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_saveConfig(t *testing.T) {
	type args struct {
		path   string
		config map[string]interface{}
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
			if err := saveConfig(tt.args.path, tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("saveConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_setConfigPropertyQuiet(t *testing.T) {
	type args struct {
		propName  string
		propValue string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setConfigPropertyQuiet(tt.args.propName, tt.args.propValue)
		})
	}
}

func Test_verifyConfigDir(t *testing.T) {
	type args struct {
		fullPath string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifyConfigDir(tt.args.fullPath)
		})
	}
}
