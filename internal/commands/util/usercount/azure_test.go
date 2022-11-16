package usercount

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func TestAzureUserCountOrgs(t *testing.T) {
	cmd := NewUserCountCommand(nil, mock.AzureMockWrapper{}, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Azure user count command must exist")

	cmd.SetArgs(
		[]string{
			AzureCommand,
			"--" + OrgsFlag,
			"a,b",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil)
}

func TestAzureUserCountRepos(t *testing.T) {
	cmd := NewUserCountCommand(nil, mock.AzureMockWrapper{}, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Azure user count command must exist")

	cmd.SetArgs(
		[]string{
			AzureCommand,
			"--" + OrgsFlag,
			"a",
			"--" + projectFlag,
			"a",
			"--" + ReposFlag,
			"a,b",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil)
}

func TestAzureUserCountProjects(t *testing.T) {
	cmd := NewUserCountCommand(nil, mock.AzureMockWrapper{}, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Azure user count command must exist")

	cmd.SetArgs(
		[]string{
			AzureCommand,
			"--" + OrgsFlag,
			"a",
			"--" + projectFlag,
			"a,b,c",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil)
}

func TestAzureCountMissingArgs(t *testing.T) {
	cmd := NewUserCountCommand(nil, mock.AzureMockWrapper{}, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Azure user count command must exist")

	cmd.SetArgs(
		[]string{
			AzureCommand,
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingOrganization)
}

func TestAzureCountMissingOrgs(t *testing.T) {
	cmd := NewUserCountCommand(nil, mock.AzureMockWrapper{}, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Azure user count command must exist")

	cmd.SetArgs(
		[]string{
			AzureCommand,
			"--" + projectFlag,
			"a",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingOrganization)
}

func TestAzureCountMissingProject(t *testing.T) {
	cmd := NewUserCountCommand(nil, mock.AzureMockWrapper{}, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Azure user count command must exist")

	cmd.SetArgs(
		[]string{
			AzureCommand,
			"--" + OrgsFlag,
			"a",
			"--" + ReposFlag,
			"a,b",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingProject)
}

func TestAzureCountMultipleOrgsWithRepo(t *testing.T) {
	cmd := NewUserCountCommand(nil, mock.AzureMockWrapper{}, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Azure user count command must exist")

	cmd.SetArgs(
		[]string{
			AzureCommand,
			"--" + OrgsFlag,
			"a,b",
			"--" + projectFlag,
			"a",
			"--" + ReposFlag,
			"a,b",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, azureManyOrgsOnRepo)
}

func TestAzureUserCountOrgsUrl(t *testing.T) {
	cmd := NewUserCountCommand(nil, mock.AzureMockWrapper{}, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Azure user count command must exist")

	cmd.SetArgs(
		[]string{
			AzureCommand,
			"--" + OrgsFlag,
			"a,b",
			"--" + url,
			"a",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil)
}
