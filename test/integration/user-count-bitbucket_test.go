//go:build integrationzz

package integration

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/commands/util/usercount"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

const (
	envWorkspace      = "BITBUCKET_WORKSPACE"
	envBitBucketRepos = "BITBUCKET_REPOS"
	envUsername       = "BITBUCKET_USERNAME"
	envPassword       = "BITBUCKET_PASSWORD"
	workspaceFlag     = "workspaces"
)

func TestBitbucketUserCountWorkspace(t *testing.T) {
	_ = viper.BindEnv(pat)
	buffer := executeCmdNilAssertion(
		t,
		"Counting contributors from checkmarxdev should pass",
		"utils",
		usercount.UcCommand,
		usercount.BitBucketCommand,
		flag(workspaceFlag),
		os.Getenv(envWorkspace),
		flag(params.UsernameFlag),
		os.Getenv(envUsername),
		flag(params.PasswordFlag),
		os.Getenv(envPassword),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)

	var parsedJson []usercount.RepositoryView
	scanner := bufio.NewScanner(buffer)
	for scanner.Scan() {
		json.Unmarshal([]byte(scanner.Text()), &parsedJson)
		break
	}
	totalView := parsedJson[len(parsedJson)-1]
	assert.Assert(t, len(parsedJson) >= 1)
	assert.Assert(t, totalView.Name == usercount.TotalContributorsName)
	assert.Assert(t, totalView.UniqueContributors >= 0)
}

func TestBitbucketUserCountRepos(t *testing.T) {
	_ = viper.BindEnv(pat)
	buffer := executeCmdNilAssertion(
		t,
		"Counting contributors from checkmarxdev should pass",
		"utils",
		usercount.UcCommand,
		usercount.BitBucketCommand,
		flag(workspaceFlag),
		os.Getenv(envWorkspace),
		flag(usercount.ReposFlag),
		os.Getenv(envBitBucketRepos),
		flag(params.UsernameFlag),
		os.Getenv(envUsername),
		flag(params.PasswordFlag),
		os.Getenv(envPassword),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)

	var parsedJson []usercount.RepositoryView
	scanner := bufio.NewScanner(buffer)
	for scanner.Scan() {
		json.Unmarshal([]byte(scanner.Text()), &parsedJson)
		break
	}
	totalView := parsedJson[len(parsedJson)-1]
	assert.Assert(t, len(parsedJson) >= 1)
	assert.Assert(t, totalView.Name == usercount.TotalContributorsName)
	assert.Assert(t, totalView.UniqueContributors >= 0)
}

func TestBitbucketUserCountReposDebug(t *testing.T) {
	_ = viper.BindEnv(pat)
	buffer := executeCmdNilAssertion(
		t,
		"Counting contributors from checkmarxdev should pass",
		"utils",
		usercount.UcCommand,
		usercount.BitBucketCommand,
		flag(workspaceFlag),
		os.Getenv(envWorkspace),
		flag(usercount.ReposFlag),
		os.Getenv(envBitBucketRepos),
		flag(params.UsernameFlag),
		os.Getenv(envUsername),
		flag(params.PasswordFlag),
		os.Getenv(envPassword),
		flag(params.FormatFlag),
		printer.FormatJSON,
		flag(params.DebugFlag),
	)

	var parsedJson []usercount.RepositoryView
	scanner := bufio.NewScanner(buffer)
	for scanner.Scan() {
		json.Unmarshal([]byte(scanner.Text()), &parsedJson)
		break
	}
	totalView := parsedJson[len(parsedJson)-1]
	assert.Assert(t, len(parsedJson) >= 1)
	assert.Assert(t, totalView.Name == usercount.TotalContributorsName)
	assert.Assert(t, totalView.UniqueContributors >= 0)
}

func TestBitbucketCountWorkspaceFailed(t *testing.T) {
	_ = viper.BindEnv(pat)
	err, _ := executeCommand(
		t,
		"utils",
		usercount.UcCommand,
		usercount.BitBucketCommand,
		flag(params.FormatFlag),
		printer.FormatJSON,
	)

	assertError(t, err, "Provide at least one workspace")
}

func TestBitbucketCountRepoFailed(t *testing.T) {
	_ = viper.BindEnv(pat)
	err, _ := executeCommand(
		t,
		"utils",
		usercount.UcCommand,
		usercount.BitBucketCommand,
		flag(usercount.ReposFlag),
		os.Getenv(envBitBucketRepos),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)

	assertError(t, err, "Provide at least one workspace")
}

func TestBitbucketCountMultipleWorkspaceFailed(t *testing.T) {
	_ = viper.BindEnv(pat)
	err, _ := executeCommand(
		t,
		"utils",
		usercount.UcCommand,
		usercount.BitBucketCommand,
		flag(workspaceFlag),
		os.Getenv(envWorkspace)+",MOCK",
		flag(usercount.ReposFlag),
		os.Getenv(envBitBucketRepos),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)

	assertError(t, err, "You must provide a single workspace for repo counting")
}
