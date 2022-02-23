package integration

import (
	"testing"

	user_count "github.com/checkmarx/ast-cli/internal/commands/util/user-count"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestGitHubUserCount(t *testing.T) {
	buffer := executeCmdNilAssertion(
		t,
		"msg",
		"utils",
		"user-count",
		"github",
		"--orgs",
		"checkmarxdev,checkmarx,checkmarx-ltd",
		"--token",
		viper.GetString("PERSONAL_ACCESS_TOKEN"),
		"--format",
		"json",
	)

	var parsedJson []user_count.RepositoryView
	unmarshall(t, buffer, &parsedJson, "Reading unique contributors json should pass")

	totalView := parsedJson[len(parsedJson)-1]
	assert.Assert(t, len(parsedJson) >= 1)
	assert.Assert(t, totalView.Name == user_count.TotalContributorsName)
	assert.Assert(t, totalView.UniqueContributors >= 0)
}
