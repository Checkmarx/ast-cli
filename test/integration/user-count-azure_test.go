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

const (
	envOrg      = "AZURE_ORG"
	envToken    = "AZURE_TOKEN"
	envProject  = "AZURE_PROJECT"
	envRepos    = "AZURE_REPOS"
	projectFlag = "projects"
)

func TestAzureUserCountOrgs(t *testing.T) {
	_ = viper.BindEnv(pat)
	buffer := executeCmdNilAssertion(
		t,
		"Counting contributors from checkmarxdev should pass",
		utilsCommand,
		usercount.UcCommand,
		usercount.AzureCommand,
		flag(usercount.OrgsFlag),
		os.Getenv(envOrg),
		flag(params.SCMTokenFlag),
		os.Getenv(envToken),
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

func TestAzureUserCountProjects(t *testing.T) {
	_ = viper.BindEnv(pat)
	buffer := executeCmdNilAssertion(
		t,
		"Counting contributors from checkmarxdev should pass",
		utilsCommand,
		usercount.UcCommand,
		usercount.AzureCommand,
		flag(usercount.OrgsFlag),
		os.Getenv(envOrg),
		flag(projectFlag),
		os.Getenv(envProject),
		flag(params.SCMTokenFlag),
		os.Getenv(envToken),
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

func TestAzureUserCountRepos(t *testing.T) {
	_ = viper.BindEnv(pat)
	buffer := executeCmdNilAssertion(
		t,
		"Counting contributors from checkmarxdev should pass",
		utilsCommand,
		usercount.UcCommand,
		usercount.AzureCommand,
		flag(usercount.OrgsFlag),
		os.Getenv(envOrg),
		flag(projectFlag),
		os.Getenv(envProject),
		flag(usercount.ReposFlag),
		os.Getenv(envRepos),
		flag(params.SCMTokenFlag),
		os.Getenv(envToken),
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

func TestAzureUserCountOrgsFailed(t *testing.T) {
	_ = viper.BindEnv(pat)
	err, _ := executeCommand(
		t,
		utilsCommand,
		usercount.UcCommand,
		usercount.AzureCommand,
		flag(params.SCMTokenFlag),
		os.Getenv(envToken),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)
	assertError(t, err, "Provide at least one organization")
}

func TestAzureUserCountReposFailed(t *testing.T) {
	_ = viper.BindEnv(pat)
	err, _ := executeCommand(
		t,
		utilsCommand,
		usercount.UcCommand,
		usercount.AzureCommand,
		flag(usercount.OrgsFlag),
		os.Getenv(envOrg),
		flag(usercount.ReposFlag),
		os.Getenv(envRepos),
		flag(params.SCMTokenFlag),
		os.Getenv(envToken),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)
	assertError(t, err, "Provide at least one project")
}

func TestAzureCountMultipleWorkspaceFailed(t *testing.T) {
	_ = viper.BindEnv(pat)
	err, _ := executeCommand(
		t,
		utilsCommand,
		usercount.UcCommand,
		usercount.AzureCommand,
		flag(usercount.OrgsFlag),
		os.Getenv(envOrg)+",MOCK",
		flag(projectFlag),
		os.Getenv(envProject),
		flag(usercount.ReposFlag),
		os.Getenv(envRepos),
		flag(params.SCMTokenFlag),
		os.Getenv(envToken),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)
	assertError(t, err, "You must provide a single org for repo counting")
}

func TestAzureUserCountWrongToken(t *testing.T) {
	_ = viper.BindEnv(pat)
	err, _ := executeCommand(
		t,
		utilsCommand,
		usercount.UcCommand,
		usercount.AzureCommand,
		flag(usercount.OrgsFlag),
		os.Getenv(envOrg),
		flag(params.SCMTokenFlag),
		"wrong",
		flag(params.FormatFlag),
		printer.FormatJSON,
	)
	assertError(t, err, "failed Azure Authentication")
}

func TestAzureUserCountWrongOrg(t *testing.T) {
	_ = viper.BindEnv(pat)
	err, _ := executeCommand(
		t,
		utilsCommand,
		usercount.UcCommand,
		usercount.AzureCommand,
		flag(usercount.OrgsFlag),
		"wrong",
		flag(params.SCMTokenFlag),
		os.Getenv(envToken),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)
	assert.ErrorContains(t, err, "unauthorized")
}
