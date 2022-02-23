//go:build integration

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/userCount"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestGitHubUserCount(t *testing.T) {
	buffer := executeCmdNilAssertion(
		t,
		"msg",
		"utils",
		"userCount",
		"github",
		"--orgs",
		"checkmarxdev,checkmarx,checkmarx-ltd",
		"--token",
		viper.GetString("PERSONAL_ACCESS_TOKEN"),
		"--format",
		"json",
	)

	var parsedJson []userCount.RepositoryView
	unmarshall(t, buffer, &parsedJson, "Reading unique contributors json should pass")

	totalView := parsedJson[len(parsedJson)-1]
	assert.Assert(t, len(parsedJson) >= 1)
	assert.Assert(t, totalView.Name == userCount.TotalContributorsName)
	assert.Assert(t, totalView.UniqueContributors >= 0)
}
