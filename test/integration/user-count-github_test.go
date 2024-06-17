//go:build integrationzz

package integration

import (
	"bufio"
	"encoding/json"
	"testing"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/commands/util/usercount"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestGitHubUserCount(t *testing.T) {
	_ = viper.BindEnv(pat)
	buffer := executeCmdWithTimeOutNilAssertion(
		t,
		"Counting contributors from checkmarx should pass",
		2*time.Minute,
		"utils",
		usercount.UcCommand,
		usercount.GithubCommand,
		flag(usercount.OrgsFlag),
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

func TestGitHubUserCountRepos(t *testing.T) {
	_ = viper.BindEnv(pat)
	buffer := executeCmdNilAssertion(
		t,
		"Counting contributors from checkmarxdev should pass",
		"utils",
		usercount.UcCommand,
		usercount.GithubCommand,
		flag(usercount.OrgsFlag),
		"checkmarxdev",
		flag(usercount.ReposFlag),
		"ast-cli-javascript-wrapper",
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

func TestGitHubUserCountFailed(t *testing.T) {
	_ = viper.BindEnv(pat)
	err, _ := executeCommand(
		t,
		"utils",
		usercount.UcCommand,
		usercount.GithubCommand,
		flag(usercount.OrgsFlag),
		"",
		flag(params.SCMTokenFlag),
		viper.GetString(pat),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)
	assertError(t, err, "provide at least one repository or organization")
}
