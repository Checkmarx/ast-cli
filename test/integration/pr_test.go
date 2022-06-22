//go:build integration

package integration

import (
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"testing"
)

func TestPRDecorationSuccessCase(t *testing.T) {
	_ = viper.BindEnv(params.SCMTokenKey)
	_ = viper.BindEnv(params.CxScanKey)
	_ = viper.BindEnv(params.OrgNamespaceKey)
	_ = viper.BindEnv(params.OrgRepoNameKey)
	_ = viper.BindEnv(params.PRNumberKey)

	args := []string{
		"utils",
		"pr",
		flag(params.ScanIDFlag),
		viper.GetString(params.CxScanKey),
		flag(params.SCMTokenFlag),
		viper.GetString(params.SCMTokenKey),
		flag(params.NamespaceFlag),
		viper.GetString(params.OrgNamespaceKey),
		flag(params.PRNumberFlag),
		viper.GetString(params.PRNumberKey),
		flag(params.RepoNameFlag),
		viper.GetString(params.OrgRepoNameKey),
	}
	err, _ := executeCommand(t, args...)
	assertError(t, err, "Response status code 201")
}

func TestPRDecorationFailure(t *testing.T) {
	_ = viper.BindEnv(params.SCMTokenKey)
	_ = viper.BindEnv(params.OrgNamespaceKey)
	_ = viper.BindEnv(params.OrgRepoNameKey)
	_ = viper.BindEnv(params.PRNumberKey)

	args := []string{
		"utils",
		"pr",
		flag(params.ScanIDFlag),
		"",
		flag(params.SCMTokenFlag),
		viper.GetString(params.SCMTokenKey),
		flag(params.NamespaceFlag),
		viper.GetString(params.OrgNamespaceKey),
		flag(params.PRNumberFlag),
		viper.GetString(params.PRNumberKey),
		flag(params.RepoNameFlag),
		viper.GetString(params.OrgRepoNameKey),
	}
	err, _ := executeCommand(t, args...)
	assertError(t, err, "Value of scan-id is invalid")
}
