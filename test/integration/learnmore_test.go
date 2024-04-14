//go:build integration

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"gotest.tools/assert"
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

func TestGetLearnMoreInformationSuccessCaseJson(t *testing.T) {
	_ = viper.BindEnv("QUERY_ID")
	queryID := viper.GetString("QUERY_ID")
	if queryID == "" {
		queryID = "1234"
	}
	err, _ := executeCommand(
		t, "utils", "learn-more",
		flag(params.QueryIDFlag), queryID,
		flag(params.FormatFlag), "json")
	assert.NilError(t, err, "Must not fail")
}

func TestGetLearnMoreInformationSuccessCaseConsole(t *testing.T) {
	_ = viper.BindEnv("QUERY_ID")
	queryID := viper.GetString("QUERY_ID")
	if queryID == "" {
		queryID = "1234"
	}
	err, _ := executeCommand(
		t, "utils", "learn-more",
		flag(params.QueryIDFlag), queryID)
	assert.NilError(t, err, "Must not fail")
}
