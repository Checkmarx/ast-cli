package usercount

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func TestBitbucketUserCountWorkspace(t *testing.T) {
	cmd := NewUserCountCommand(nil, nil, mock.BitBucketMockWrapper{}, nil, nil)
	assert.Assert(t, cmd != nil, "BitBucket user count command must exist")

	cmd.SetArgs(
		[]string{
			BitBucketCommand,
			"--" + workspaceFlag,
			"a",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil)
}

func TestBitbucketUserCountRepos(t *testing.T) {
	cmd := NewUserCountCommand(nil, nil, mock.BitBucketMockWrapper{}, nil, nil)
	assert.Assert(t, cmd != nil, "BitBucket user count command must exist")

	cmd.SetArgs(
		[]string{
			BitBucketCommand,
			"--" + workspaceFlag,
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

func TestBitbucketUserCountWorkspaceFailed(t *testing.T) {
	cmd := NewUserCountCommand(nil, nil, mock.BitBucketMockWrapper{}, nil, nil)
	assert.Assert(t, cmd != nil, "BitBucket user count command must exist")

	cmd.SetArgs(
		[]string{
			BitBucketCommand,
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingWorkspace)
}

func TestBitbucketUserCountRepoFailed(t *testing.T) {
	cmd := NewUserCountCommand(nil, nil, mock.BitBucketMockWrapper{}, nil, nil)
	assert.Assert(t, cmd != nil, "BitBucket user count command must exist")

	cmd.SetArgs(
		[]string{
			BitBucketCommand,
			"--" + ReposFlag,
			"a,b",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingWorkspace)
}

func TestBitBucketCountMultipleOrgsWithRepo(t *testing.T) {
	cmd := NewUserCountCommand(nil, nil, mock.BitBucketMockWrapper{}, nil, nil)
	assert.Assert(t, cmd != nil, "BitBucket user count command must exist")

	cmd.SetArgs(
		[]string{
			BitBucketCommand,
			"--" + workspaceFlag,
			"a,b",
			"--" + ReposFlag,
			"c",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, bitbucketManyWorkspaceOnRepo)
}
