//go:build integration

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
