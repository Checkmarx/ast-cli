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
	"gotest.tools/assert"
)

const (
	gitLabEnvToken = "GITLAB_TOKEN"
)

func TestGitLabUserCountOnlyUserProjects(t *testing.T) {
	buffer := executeCmdNilAssertion(
		t, "Counting contributors from gitlab user projects should pass",
		"utils", usercount.UcCommand,
		usercount.GitLabCommand,
		flag(params.SCMTokenFlag), os.Getenv(gitLabEnvToken),
		flag(params.FormatFlag), printer.FormatJSON,
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

func TestGitLabUserCountOnlyGroup(t *testing.T) {
	buffer := executeCmdNilAssertion(
		t, "Counting contributors from gitlab group should pass",
		"utils", usercount.UcCommand,
		usercount.GitLabCommand,
		flag(usercount.GitLabGroupsFlag), "gitlab-org/quality",
		flag(params.SCMTokenFlag), os.Getenv(gitLabEnvToken),
		flag(params.FormatFlag), printer.FormatJSON,
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

func TestGitLabUserCountOnlyProject(t *testing.T) {
	buffer := executeCmdNilAssertion(
		t, "Counting contributors from gitlab project should pass",
		"utils", usercount.UcCommand,
		usercount.GitLabCommand,
		flag(usercount.GitLabProjectsFlag), "gitlab-org/gitlab",
		flag(params.SCMTokenFlag), os.Getenv(gitLabEnvToken),
		flag(params.FormatFlag), printer.FormatJSON,
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

func TestGitLabUserCountBothProjectAndGroup(t *testing.T) {
	err, _ := executeCommand(
		t,
		"utils", usercount.UcCommand, usercount.GitLabCommand,
		flag(usercount.GitLabGroupsFlag), "checkmarx",
		flag(usercount.GitLabProjectsFlag), "cxlite",
		flag(params.SCMTokenFlag), os.Getenv(gitLabEnvToken),
		flag(params.FormatFlag), printer.FormatJSON,
	)

	assertError(t, err, "Projects and Groups both cannot be provided at the same time.")
}

func TestGitLabUserCountInvalidProject(t *testing.T) {
	err, _ := executeCommand(
		t,
		"utils", usercount.UcCommand, usercount.GitLabCommand,
		flag(usercount.GitLabProjectsFlag), "/ab12xy!@#",
		flag(params.SCMTokenFlag), os.Getenv(gitLabEnvToken),
		flag(params.FormatFlag), printer.FormatJSON,
	)

	assertError(t, err, "{\"message\":\"404 Project Not Found\"}")
}

func TestGitLabUserCountInvalidGroup(t *testing.T) {
	err, _ := executeCommand(
		t,
		"utils", usercount.UcCommand, usercount.GitLabCommand,
		flag(usercount.GitLabGroupsFlag), "/ab12xy!@#",
		flag(params.SCMTokenFlag), os.Getenv(gitLabEnvToken),
		flag(params.FormatFlag), printer.FormatJSON,
	)

	assertError(t, err, "{\"message\":\"404 Group Not Found\"}")
}

func TestGitLabUserCountBlankGroupValue(t *testing.T) {
	buffer := executeCmdNilAssertion(
		t, "Counting contributors from gitlab group should pass",
		"utils", usercount.UcCommand,
		usercount.GitLabCommand,
		flag(usercount.GitLabGroupsFlag), " ",
		flag(params.SCMTokenFlag), os.Getenv(gitLabEnvToken),
		flag(params.FormatFlag), printer.FormatJSON,
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

func TestGitLabUserCountBlankProjectValue(t *testing.T) {
	buffer := executeCmdNilAssertion(
		t, "Counting contributors from gitlab group should pass",
		"utils", usercount.UcCommand,
		usercount.GitLabCommand,
		flag(usercount.GitLabProjectsFlag), " ",
		flag(params.SCMTokenFlag), os.Getenv(gitLabEnvToken),
		flag(params.FormatFlag), printer.FormatJSON,
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
