//go:build integrationzz

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

func TestGetTenantConfigurationSuccessCaseJson(t *testing.T) {
	err, _ := executeCommand(
		t, "utils", "tenant",
		flag(params.FormatFlag), "json",
	)
	assert.NilError(t, err, "Must not fail")
}

func TestGetTenantConfigurationSuccessCaseList(t *testing.T) {
	err, _ := executeCommand(t, "utils", "tenant")
	assert.NilError(t, err, "Must not fail")
}

//func TestGetTenantConfigurationFailCase(t *testing.T) {
//
//	defaultValue := viper.GetString(params.TenantConfigurationPathKey)
//	go func() {
//		viper.Set(params.TenantConfigurationPathKey, defaultValue)
//	}()
//
//	viper.Set(params.TenantConfigurationPathEnv, "api/scans")
//
//	err, _ := executeCommand(t, "utils", "tenant")
//	assert.Assert(t, err != nil)
//
//	viper.Set(params.TenantConfigurationPathEnv, "api/notfound")
//
//	err, _ = executeCommand(t, "utils", "tenant")
//	assert.Assert(t, err != nil)
//}
