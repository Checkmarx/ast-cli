//go:build integration

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
		"Counting contributors from checkmarxdev should pass",
		2*time.Minute,
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
