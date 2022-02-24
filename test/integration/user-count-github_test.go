//go:build integration

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/usercount"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestGitHubUserCount(t *testing.T) {
	_ = viper.BindEnv("PERSONAL_ACCESS_TOKEN")
	buffer := executeCmdNilAssertion(
		t,
		"msg",
		"utils",
		"user-count",
		"github",
		"--orgs",
		"checkmarxdev",
		"--token",
		viper.GetString(pat),
		"--format",
		"json",
	)

	var parsedJson []usercount.RepositoryView
	unmarshall(t, buffer, &parsedJson, "Reading unique contributors json should pass")

	totalView := parsedJson[len(parsedJson)-1]
	assert.Assert(t, len(parsedJson) >= 1)
	assert.Assert(t, totalView.Name == usercount.TotalContributorsName)
	assert.Assert(t, totalView.UniqueContributors >= 0)
}
