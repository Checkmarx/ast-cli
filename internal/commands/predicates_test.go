//go:build !integration

package commands

import (
	"fmt"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
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

func TestTriageGetStatesFlag(t *testing.T) {
	mockWrapper := &mock.CustomStatesMockWrapper{}
	cmd := triageGetStatesSubCommand(mockWrapper)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.NilError(t, err)
	states, err := mockWrapper.GetAllCustomStates(false)
	assert.NilError(t, err)
	assert.Equal(t, len(states), 2)
	cmd.SetArgs([]string{"--all"})
	err = cmd.Execute()
	assert.NilError(t, err)
	states, err = mockWrapper.GetAllCustomStates(true)
	assert.NilError(t, err)
	assert.Equal(t, len(states), 3)
}