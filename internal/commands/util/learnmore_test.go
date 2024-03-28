//go:build integration

package util

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"

	"gotest.tools/assert"
)

func TestLearnMoreHelp(t *testing.T) {
	cmd := NewLearnMoreCommand(nil)
	cmd.SetArgs([]string{"utils", "learn-more", "--help"})
	err := cmd.Execute()
	assert.Assert(t, err == nil)
}

func TestLearnMoreMockQueryIdSummaryConsole(t *testing.T) {
	cmd := NewLearnMoreCommand(mock.LearnMoreMockWrapper{})
	cmd.SetArgs([]string{"utils", "learn-more", "--query-id", "MOCK"})
	err := cmd.Execute()
	assert.NilError(t, err, "Learn more command should run with no errors and print to console")
}

func TestLearnMoreMockQueryIdJsonFormat(t *testing.T) {
	cmd := NewLearnMoreCommand(mock.LearnMoreMockWrapper{})
	cmd.SetArgs([]string{"utils", "learn-more", "--query-id", "MOCK", "--format", "json"})
	err := cmd.Execute()
	assert.NilError(t, err, "Learn more command should run with no errors and print to json")
}

func TestLearnMoreMockQueryIdListFormat(t *testing.T) {
	cmd := NewLearnMoreCommand(mock.LearnMoreMockWrapper{})
	cmd.SetArgs([]string{"utils", "learn-more", "--query-id", "MOCK", "--format", "list"})
	err := cmd.Execute()
	assert.NilError(t, err, "Learn more command should run with no errors and print to list")
}

func TestLearnMoreMockQueryIdTableFormat(t *testing.T) {
	cmd := NewLearnMoreCommand(mock.LearnMoreMockWrapper{})
	cmd.SetArgs([]string{"utils", "learn-more", "--query-id", "MOCK", "--format", "table"})
	err := cmd.Execute()
	assert.NilError(t, err, "Learn more command should run with no errors and print to table")
}

func TestLearnMoreMockMissingQueryId(t *testing.T) {
	cmd := NewLearnMoreCommand(mock.LearnMoreMockWrapper{})
	cmd.SetArgs([]string{"utils", "learn-more"})
	err := cmd.Execute()
	assert.Assert(t, err.Error() == "required flag(s) \"query-id\" not set")
}

func TestLearnMoreMockQueryIdInvalidFormat(t *testing.T) {
	cmd := NewLearnMoreCommand(mock.LearnMoreMockWrapper{})
	cmd.SetArgs([]string{"utils", "learn-more", "--query-id", "MOCK", "--format", "MOCK"})
	err := cmd.Execute()
	assert.Assert(t, err.Error() == mockFormatErrorMessage)
}

func TestCommand(t *testing.T) {
	cmd := NewLearnMoreCommand(nil)
	assert.Assert(t, cmd != nil, "Learnmore command must exist")
}
