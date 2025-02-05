//go:build !integration

package commands

import (
	"fmt"
	"github.com/checkmarx/ast-cli/internal/wrappers"
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

func TestGetCustomStateID(t *testing.T) {
	tests := []struct {
		name                string
		state               string
		mockWrapper         wrappers.CustomStatesWrapper
		expectedStateID     string
		expectedErrorString string
	}{
		{
			name:            "State found",
			state:           "demo3",
			mockWrapper:     &mock.CustomStatesMockWrapper{},
			expectedStateID: "3",
		},
		{
			name:                "State not found",
			state:               "nonexistent",
			mockWrapper:         &mock.CustomStatesMockWrapper{},
			expectedStateID:     "",
			expectedErrorString: "No matching state found for nonexistent",
		},
		{
			name:                "Error fetching states",
			state:               "nonexistent",
			mockWrapper:         &mock.CustomStatesMockWrapperWithError{},
			expectedStateID:     "",
			expectedErrorString: "Failed to fetch custom states",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			stateID, err := getCustomStateID(tt.mockWrapper, tt.state)
			if tt.expectedErrorString != "" {
				assert.ErrorContains(t, err, tt.expectedErrorString)
			} else {
				assert.NilError(t, err)
			}
			assert.Equal(t, stateID, tt.expectedStateID)
		})
	}
}

func TestIsCustomState(t *testing.T) {
	tests := []struct {
		state    string
		isCustom bool
	}{
		{"TO_VERIFY", false},
		{"to_verify", false},
		{"NOT_EXPLOITABLE", false},
		{"PROPOSED_NOT_EXPLOITABLE", false},
		{"CONFIRMED", false},
		{"URGENT", false},
		{"CUSTOM_STATE_1", true},
		{"CUSTOM_STATE_2", true},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := isCustomState(tt.state)
			assert.Equal(t, result, tt.isCustom)
		})
	}
}
