package util

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func TestTenantConfigurationHelp(t *testing.T) {
	cmd := NewTenantConfigurationCommand(mock.TenantConfigurationMockWrapper{})
	cmd.SetArgs([]string{"utils", "tenant", "--help"})
	err := cmd.Execute()
	assert.Assert(t, err == nil)
}

func TestTenantConfigurationJsonFormat(t *testing.T) {
	cmd := NewTenantConfigurationCommand(mock.TenantConfigurationMockWrapper{})
	cmd.SetArgs([]string{"utils", "tenant", "--format", "json"})
	err := cmd.Execute()
	assert.NilError(t, err, "Tenant configuration command should run with no errors and print to json")
}

func TestTenantConfigurationListFormat(t *testing.T) {
	cmd := NewTenantConfigurationCommand(mock.TenantConfigurationMockWrapper{})
	cmd.SetArgs([]string{"utils", "tenant", "--format", "list"})
	err := cmd.Execute()
	assert.NilError(t, err, "Tenant configuration command should run with no errors and print to list")
}

func TestTenantConfigurationTableFormat(t *testing.T) {
	cmd := NewTenantConfigurationCommand(mock.TenantConfigurationMockWrapper{})
	cmd.SetArgs([]string{"utils", "tenant", "--format", "table"})
	err := cmd.Execute()
	assert.NilError(t, err, "Tenant configuration command should run with no errors and print to table")
}

func TestTenantConfigurationInvalidFormat(t *testing.T) {
	cmd := NewTenantConfigurationCommand(mock.TenantConfigurationMockWrapper{})
	cmd.SetArgs([]string{"utils", "tenant", "--format", "MOCK"})
	err := cmd.Execute()
	assert.Assert(t, err.Error() == mockFormatErrorMessage)
}

func TestNewTenantConfigurationCommand(t *testing.T) {
	cmd := NewTenantConfigurationCommand(mock.TenantConfigurationMockWrapper{})
	assert.Assert(t, cmd != nil, "Tenant configuration command must exist")
}
