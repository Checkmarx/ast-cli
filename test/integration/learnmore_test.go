//go:build integration

package integration

import (
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"gotest.tools/assert"
	"testing"
)

func TestGetLearnMoreInformationFailure(t *testing.T) {
	err, _ := executeCommand(
		t, "utils", "learn-more",
		flag(params.QueryIDFlag), "abcd",
		flag(params.FormatFlag), "json")
	assertError(t, err, "Failed getting additional details")
}

func TestGetLearnMoreInformationFailureMissingQueryId(t *testing.T) {
	err, _ := executeCommand(
		t, "utils", "learn-more",
		flag(params.FormatFlag), "json")
	assertError(t, err, "required flag(s) \"query-id\" not set")
}

func TestGetLearnMoreInformationSuccessCase(t *testing.T) {
	_ = viper.BindEnv("QUERY_ID")
	err, _ := executeCommand(
		t, "utils", "learn-more",
		flag(params.QueryIDFlag), viper.GetString("QUERY_ID"),
		flag(params.FormatFlag), "json")
	assert.NilError(t, err, "Must not fail")
}
