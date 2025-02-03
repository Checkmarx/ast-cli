//go:build !integration

package commands

import (
	"fmt"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"testing"

	"gotest.tools/assert"
)

func TestTriageHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "triage")
}

func TestRunShowTriageCommand(t *testing.T) {
	execCmdNilAssertion(t, "triage", "show", "--project-id", "MOCK", "--similarity-id", "MOCK", "--scan-type", "sast")
}

func TestRunUpdateTriageCommand(t *testing.T) {
	execCmdNilAssertion(
		t,
		"triage",
		"update",
		"--project-id",
		"MOCK",
		"--similarity-id",
		"MOCK",
		"--state",
		"confirmed",
		"--comment",
		"Testing commands.",
		"--severity",
		"low",
		"--scan-type",
		"sast")
}

func TestRunShowTriageCommandWithNoInput(t *testing.T) {
	err := execCmdNotNilAssertion(t, "triage", "show")
	assert.Assert(t, err.Error() == "required flag(s) \"project-id\", \"scan-type\", \"similarity-id\" not set")
}

func TestRunUpdateTriageCommandWithNoInput(t *testing.T) {
	err := execCmdNotNilAssertion(t, "triage", "update")
	fmt.Println(err)
	assert.Assert(
		t,
		err.Error() == "required flag(s) \"project-id\", \"scan-type\", \"severity\", \"similarity-id\", \"state\" not set")
}

type MockCustomStatesWrapper struct {
	ReceivedIncludeDeleted bool
}

func (m *MockCustomStatesWrapper) GetAllCustomStates(includeDeleted bool) ([]wrappers.CustomState, error) {
	m.ReceivedIncludeDeleted = includeDeleted
	return []wrappers.CustomState{
		{ID: 1, Name: "State1", Type: "Custom"},
		{ID: 2, Name: "State2", Type: "System"},
	}, nil
}

func TestTriageGetStatesFlag(t *testing.T) {
	mockWrapper := &MockCustomStatesWrapper{}
	cmd := triageGetStatesSubCommand(mockWrapper)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.NilError(t, err)
	assert.Equal(t, mockWrapper.ReceivedIncludeDeleted, false)
	cmd.SetArgs([]string{"--all"})
	err = cmd.Execute()
	assert.NilError(t, err)
	assert.Equal(t, mockWrapper.ReceivedIncludeDeleted, true)
}