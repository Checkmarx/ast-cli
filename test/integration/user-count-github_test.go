//go:build integration

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/commands/util/usercount"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestGitHubUserCount(t *testing.T) {
	_ = viper.BindEnv(pat)
	buffer := executeCmdNilAssertion(
		t,
		"Counting contributors from checkmarxdev should pass",
		"utils",
		usercount.UcCommand,
		usercount.GithubCommand,
		flag(usercount.OrgsFlag),
		"checkmarxdev",
		flag(params.SCMTokenFlag),
		viper.GetString(pat),
		flag(params.FormatFlag),
		printer.FormatJSON,
	)

	var parsedJson []usercount.RepositoryView
	unmarshall(t, buffer, &parsedJson, "Reading unique contributors json should pass")

	totalView := parsedJson[len(parsedJson)-1]
	assert.Assert(t, len(parsedJson) >= 1)
	assert.Assert(t, totalView.Name == usercount.TotalContributorsName)
	assert.Assert(t, totalView.UniqueContributors >= 0)
}
