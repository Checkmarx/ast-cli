//go:build !integration

package dast

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func createDastScansTestCommand(args ...string) error {
	cmd := NewDastScansCommand(&mock.DastScansMockWrapper{})
	cmd.SetArgs(args)
	return cmd.Execute()
}

func TestDastScansHelp(t *testing.T) {
	err := createDastScansTestCommand("--help")
	assert.NilError(t, err)
}

func TestDastScansNoSub(t *testing.T) {
	err := createDastScansTestCommand()
	assert.NilError(t, err)
}

func TestDastScansList(t *testing.T) {
	err := createDastScansTestCommand("list", "--environment-id", "test-env-id")
	assert.NilError(t, err)
}

func TestDastScansListWithFormat(t *testing.T) {
	err := createDastScansTestCommand("list", "--environment-id", "test-env-id", "--format", "json")
	assert.NilError(t, err)
}

func TestDastScansListWithFilters(t *testing.T) {
	err := createDastScansTestCommand("list", "--environment-id", "test-env-id", "--filter", "from=1,to=10")
	assert.NilError(t, err)
}

func TestDastScansListWithSearch(t *testing.T) {
	err := createDastScansTestCommand("list", "--environment-id", "test-env-id", "--filter", "search=test")
	assert.NilError(t, err)
}

func TestDastScansListWithSort(t *testing.T) {
	err := createDastScansTestCommand("list", "--environment-id", "test-env-id", "--filter", "sort=created:desc")
	assert.NilError(t, err)
}

func TestDastScansListMissingEnvironmentID(t *testing.T) {
	err := createDastScansTestCommand("list")
	assert.ErrorContains(t, err, "required flag")
}

