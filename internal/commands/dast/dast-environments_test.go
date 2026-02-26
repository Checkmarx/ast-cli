//go:build !integration

package dast

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func createTestCommand(args ...string) error {
	cmd := NewDastEnvironmentsCommand(&mock.DastEnvironmentsMockWrapper{})
	cmd.SetArgs(args)
	return cmd.Execute()
}

func TestDastEnvironmentsHelp(t *testing.T) {
	err := createTestCommand("--help")
	assert.NilError(t, err)
}

func TestDastEnvironmentsNoSub(t *testing.T) {
	err := createTestCommand()
	assert.NilError(t, err)
}

func TestDastEnvironmentsList(t *testing.T) {
	err := createTestCommand("list")
	assert.NilError(t, err)
}

func TestDastEnvironmentsListWithFormat(t *testing.T) {
	err := createTestCommand("list", "--format", "json")
	assert.NilError(t, err)
}

func TestDastEnvironmentsListWithFilters(t *testing.T) {
	err := createTestCommand("list", "--filter", "from=1,to=10")
	assert.NilError(t, err)
}

func TestDastEnvironmentsListWithSearch(t *testing.T) {
	err := createTestCommand("list", "--filter", "search=test")
	assert.NilError(t, err)
}

func TestDastEnvironmentsListWithSort(t *testing.T) {
	err := createTestCommand("list", "--filter", "sort=domain:asc")
	assert.NilError(t, err)
}
