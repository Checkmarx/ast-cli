//go:build integration

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

func TestGitLabUserCount(t *testing.T) {
	_ = viper.BindEnv(pat)
	buffer := executeCmdNilAssertion(
		t,
		"Counting contributors from checkmarxdev should pass",
		"utils",
		usercount.UcCommand,
		usercount.GitLabCommand,
		flag(usercount.gitLabGroupsFlag),
		"checkmarx",
		flag(params.SCMTokenFlag),
		viper.GetString(pat),
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

func TestGitLabUserCountFailed(t *testing.T) {
	_ = viper.BindEnv(pat)
	err, _ := executeCommand(
		t,
		"utils",
		usercount.UcCommand,
		usercount.GitLabCommand,
		flag(usercount.gitLabGroupsFlag),
		os.Getenv(envOrg),
		flag(usercount.gitLabProjectsFlag),
		os.Getenv(envRepos),
		flag(params.SCMTokenFlag),
		os.Getenv(envToken),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)

	assertError(t, err, "Projects and Groups both cannot be provided at the same time.")
}
