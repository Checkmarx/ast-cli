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
	"gotest.tools/assert"
)

const (
	gitLabEnvToken = "GITLAB_TOKEN"
)

func TestGitLabUserCountOnlyGroup(t *testing.T) {
	buffer := executeCmdNilAssertion(
		t, "Counting contributors from gitlab group should pass",
		"utils", usercount.UcCommand,
		usercount.GitLabCommand,
		flag(usercount.GitLabGroupsFlag), "gitlab-org",
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

func TestGitLabUserCountNoProjectFound(t *testing.T) {
	err, _ := executeCommand(
		t,
		"utils", usercount.UcCommand, usercount.GitLabCommand,
		flag(usercount.GitLabProjectsFlag), "/ab12xy!@#",
		flag(params.SCMTokenFlag), os.Getenv(gitLabEnvToken),
		flag(params.FormatFlag), printer.FormatJSON,
	)

	assertError(t, err, "{\"message\":\"404 Project Not Found\"}")
}
