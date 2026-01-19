//go:build !integration

package dast

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func createAlertsTestCommand(args ...string) error {
	cmd := NewDastAlertsCommand(&mock.DastAlertsMockWrapper{})
	cmd.SetArgs(args)
	return cmd.Execute()
}

func TestDastAlertsHelp(t *testing.T) {
	err := createAlertsTestCommand("--help")
	assert.NilError(t, err)
}

func TestDastAlertsNoSub(t *testing.T) {
	err := createAlertsTestCommand()
	assert.NilError(t, err)
}

func TestDastAlertsList(t *testing.T) {
	err := createAlertsTestCommand("list", "--environment-id", "test-env", "--scan-id", "test-scan")
	assert.NilError(t, err)
}

func TestDastAlertsListWithFormat(t *testing.T) {
	err := createAlertsTestCommand("list", "--environment-id", "test-env", "--scan-id", "test-scan", "--format", "json")
	assert.NilError(t, err)
}

func TestDastAlertsListWithPagination(t *testing.T) {
	err := createAlertsTestCommand("list", "--environment-id", "test-env", "--scan-id", "test-scan", "--filter", "page=1,per_page=10")
	assert.NilError(t, err)
}

func TestDastAlertsListWithSearch(t *testing.T) {
	err := createAlertsTestCommand("list", "--environment-id", "test-env", "--scan-id", "test-scan", "--filter", "search=PII")
	assert.NilError(t, err)
}

func TestDastAlertsListWithSort(t *testing.T) {
	err := createAlertsTestCommand("list", "--environment-id", "test-env", "--scan-id", "test-scan", "--filter", "sort_by=severity:desc")
	assert.NilError(t, err)
}

func TestDastAlertsListMissingEnvironmentID(t *testing.T) {
	err := createAlertsTestCommand("list", "--scan-id", "test-scan")
	assert.ErrorContains(t, err, "required flag")
}

func TestDastAlertsListMissingScanID(t *testing.T) {
	err := createAlertsTestCommand("list", "--environment-id", "test-env")
	assert.ErrorContains(t, err, "required flag")
}
