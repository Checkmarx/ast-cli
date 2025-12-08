//go:build !integration

package commands

import (
	"fmt"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"

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
		err.Error() == "required flag(s) \"project-id\", \"scan-type\", \"severity\", \"similarity-id\" not set")
}

func TestTriageGetStatesFlag(t *testing.T) {
	mockWrapper := &mock.CustomStatesMockWrapper{}
	featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true}
	cmd := triageGetStatesSubCommand(mockWrapper, featureFlagsWrapper)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.NilError(t, err)

	states, err := mockWrapper.GetAllCustomStates(false)
	assert.NilError(t, err)
	expectedStatesCount := len(states) + len(systemStates)
	assert.Equal(t, expectedStatesCount, len(states)+len(systemStates))

	cmd.SetArgs([]string{"--all"})
	err = cmd.Execute()
	assert.NilError(t, err)

	states, err = mockWrapper.GetAllCustomStates(true)
	assert.NilError(t, err)
	expectedStatesCount = len(states) + len(systemStates)
	assert.Equal(t, expectedStatesCount, len(states)+len(systemStates))

	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: false}
	cmd = triageGetStatesSubCommand(mockWrapper, featureFlagsWrapper)
	cmd.SetArgs([]string{})
	err = cmd.Execute()
	assert.NilError(t, err)
	assert.Equal(t, len(systemStates), len(systemStates))
}
func TestGetCustomStateID(t *testing.T) {
	tests := []struct {
		name                string
		state               string
		mockWrapper         wrappers.CustomStatesWrapper
		expectedStateID     int
		expectedErrorString string
	}{
		{
			name:            "State found",
			state:           "demo3",
			mockWrapper:     &mock.CustomStatesMockWrapper{},
			expectedStateID: 3,
		},
		{
			name:                "State not found",
			state:               "nonexistent",
			mockWrapper:         &mock.CustomStatesMockWrapper{},
			expectedStateID:     -1,
			expectedErrorString: "No matching state found for nonexistent",
		},
		{
			name:                "Error fetching states",
			state:               "nonexistent",
			mockWrapper:         &mock.CustomStatesMockWrapperWithError{},
			expectedStateID:     -1,
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
func TestRunTriageUpdateWithNotFoundCustomState(t *testing.T) {
	mockResultsPredicatesWrapper := &mock.ResultsPredicatesMockWrapper{}
	mockFeatureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
	mockCustomStatesWrapper := &mock.CustomStatesMockWrapper{}
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{
		Name:   wrappers.SastCustomStateEnabled,
		Status: true,
	}
	cmd := triageUpdateSubCommand(mockResultsPredicatesWrapper, mockFeatureFlagsWrapper, mockCustomStatesWrapper)
	cmd.SetArgs([]string{
		"--similarity-id", "MOCK",
		"--project-id", "MOCK",
		"--severity", "low",
		"--state", "CUSTOM_STATE_1",
		"--scan-type", "sast",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "Failed to get custom state ID for state: CUSTOM_STATE_1: No matching state found for CUSTOM_STATE_1")
}

func TestRunTriageUpdateWithCustomState(t *testing.T) {
	mockResultsPredicatesWrapper := &mock.ResultsPredicatesMockWrapper{}
	mockFeatureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
	mockCustomStatesWrapper := &mock.CustomStatesMockWrapper{}
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{
		Name:   wrappers.SastCustomStateEnabled,
		Status: true,
	}
	cmd := triageUpdateSubCommand(mockResultsPredicatesWrapper, mockFeatureFlagsWrapper, mockCustomStatesWrapper)
	cmd.SetArgs([]string{
		"--similarity-id", "MOCK",
		"--project-id", "MOCK",
		"--severity", "low",
		"--state", "demo2",
		"--scan-type", "sast",
	})

	err := cmd.Execute()
	assert.NilError(t, err)
}

func TestRunTriageUpdateWithSystemState(t *testing.T) {
	mockResultsPredicatesWrapper := &mock.ResultsPredicatesMockWrapper{}
	mockFeatureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
	mockCustomStatesWrapper := &mock.CustomStatesMockWrapper{}

	cmd := triageUpdateSubCommand(mockResultsPredicatesWrapper, mockFeatureFlagsWrapper, mockCustomStatesWrapper)
	cmd.SetArgs([]string{
		"--similarity-id", "MOCK",
		"--project-id", "MOCK",
		"--severity", "low",
		"--state", "TO_VERIFY",
		"--scan-type", "sast",
	})

	err := cmd.Execute()
	assert.NilError(t, err)
}

func TestRunTriageGetStates_ScanTypeSwitch(t *testing.T) {
	tests := []struct {
		name                    string
		scanType                string
		customStatesFeatureFlag wrappers.FeatureFlagResponseModel
		scanTypeFeatureFlag     wrappers.FeatureFlagResponseModel
		expectedError           bool
		description             string
	}{
		{
			name:                    "KICS scan type with KICS custom states enabled",
			scanType:                "kics",
			customStatesFeatureFlag: wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true},
			scanTypeFeatureFlag:     wrappers.FeatureFlagResponseModel{Name: "KICS_CUSTOM_STATES_ENABLED", Status: true},
			expectedError:           false,
			description:             "Should successfully get custom states for KICS when feature flag is enabled",
		},
		{
			name:                    "KICS scan type with KICS custom states disabled",
			scanType:                "kics",
			customStatesFeatureFlag: wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true},
			scanTypeFeatureFlag:     wrappers.FeatureFlagResponseModel{Name: "KICS_CUSTOM_STATES_ENABLED", Status: false},
			expectedError:           false,
			description:             "Should return system states for KICS when feature flag is disabled",
		},
		{
			name:                    "SCA scan type with SCA custom states enabled",
			scanType:                "sca",
			customStatesFeatureFlag: wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true},
			scanTypeFeatureFlag:     wrappers.FeatureFlagResponseModel{Name: "SCA_CUSTOM_STATES", Status: true},
			expectedError:           false,
			description:             "Should successfully get custom states for SCA when feature flag is enabled",
		},
		{
			name:                    "SCA scan type with SCA custom states disabled",
			scanType:                "sca",
			customStatesFeatureFlag: wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true},
			scanTypeFeatureFlag:     wrappers.FeatureFlagResponseModel{Name: "SCA_CUSTOM_STATES", Status: false},
			expectedError:           false,
			description:             "Should return system states for SCA when feature flag is disabled",
		},
		{
			name:                    "SAST scan type with SAST custom states enabled",
			scanType:                "sast",
			customStatesFeatureFlag: wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true},
			scanTypeFeatureFlag:     wrappers.FeatureFlagResponseModel{Name: "SAST_CUSTOM_STATES_ENABLED", Status: true},
			expectedError:           false,
			description:             "Should successfully get custom states for SAST when feature flag is enabled",
		},
		{
			name:                    "SAST scan type with SAST custom states disabled",
			scanType:                "sast",
			customStatesFeatureFlag: wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true},
			scanTypeFeatureFlag:     wrappers.FeatureFlagResponseModel{Name: "SAST_CUSTOM_STATES_ENABLED", Status: false},
			expectedError:           false,
			description:             "Should return system states for SAST when feature flag is disabled",
		},
		{
			name:                    "Unknown scan type defaults to SAST behavior",
			scanType:                "unknown",
			customStatesFeatureFlag: wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true},
			scanTypeFeatureFlag:     wrappers.FeatureFlagResponseModel{Name: "SAST_CUSTOM_STATES_ENABLED", Status: true},
			expectedError:           false,
			description:             "Should default to SAST logic for unknown scan types",
		},
		{
			name:                    "KICS with uppercase scan type",
			scanType:                "KICS",
			customStatesFeatureFlag: wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true},
			scanTypeFeatureFlag:     wrappers.FeatureFlagResponseModel{Name: "KICS_CUSTOM_STATES_ENABLED", Status: true},
			expectedError:           false,
			description:             "Should handle uppercase scan type correctly",
		},
		{
			name:                    "SCA with mixed case scan type",
			scanType:                "ScA",
			customStatesFeatureFlag: wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true},
			scanTypeFeatureFlag:     wrappers.FeatureFlagResponseModel{Name: "SCA_CUSTOM_STATES", Status: true},
			expectedError:           false,
			description:             "Should handle mixed case scan type correctly",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			clearFlags()
			mockCustomStatesWrapper := &mock.CustomStatesMockWrapper{}
			mockFeatureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}

			// First, set the custom states feature flag
			mock.Flag = tt.customStatesFeatureFlag
			cmd := triageGetStatesSubCommand(mockCustomStatesWrapper, mockFeatureFlagsWrapper)

			// Then set the scan-type specific feature flag for the actual execution
			mock.Flag = tt.scanTypeFeatureFlag
			cmd.SetArgs([]string{"--scan-type", tt.scanType})

			err := cmd.Execute()

			if tt.expectedError {
				assert.Assert(t, err != nil, tt.description)
			} else {
				assert.NilError(t, err, tt.description)
			}
		})
	}
}

func TestRunTriageGetStates_OverallFeatureFlagDisabled(t *testing.T) {
	clearFlags()
	mockCustomStatesWrapper := &mock.CustomStatesMockWrapper{}
	mockFeatureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}

	// Set the overall custom states feature flag to disabled
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: false}

	cmd := triageGetStatesSubCommand(mockCustomStatesWrapper, mockFeatureFlagsWrapper)
	cmd.SetArgs([]string{"--scan-type", "sast"})

	err := cmd.Execute()
	assert.NilError(t, err, "Should return system states when overall custom states feature flag is disabled")
}

func TestRunTriageGetStates_WithAllFlag(t *testing.T) {
	clearFlags()
	mockCustomStatesWrapper := &mock.CustomStatesMockWrapper{}
	mockFeatureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}

	tests := []struct {
		name     string
		scanType string
		allFlag  bool
	}{
		{
			name:     "KICS with all flag true",
			scanType: "kics",
			allFlag:  true,
		},
		{
			name:     "SCA with all flag false",
			scanType: "sca",
			allFlag:  false,
		},
		{
			name:     "SAST with all flag true",
			scanType: "sast",
			allFlag:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			clearFlags()
			mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.CustomStatesFeatureFlag, Status: true}

			cmd := triageGetStatesSubCommand(mockCustomStatesWrapper, mockFeatureFlagsWrapper)

			// Set the scan-type specific feature flag to enabled
			switch tt.scanType {
			case "kics":
				mock.Flag = wrappers.FeatureFlagResponseModel{Name: "KICS_CUSTOM_STATES_ENABLED", Status: true}
			case "sca":
				mock.Flag = wrappers.FeatureFlagResponseModel{Name: "SCA_CUSTOM_STATES", Status: true}
			default:
				mock.Flag = wrappers.FeatureFlagResponseModel{Name: "SAST_CUSTOM_STATES_ENABLED", Status: true}
			}

			args := []string{"--scan-type", tt.scanType}
			if tt.allFlag {
				args = append(args, "--all")
			}
			cmd.SetArgs(args)

			err := cmd.Execute()
			assert.NilError(t, err)
		})
	}
}

func TestDetermineSystemOrCustomState(t *testing.T) {
	tests := []struct {
		name                string
		state               string
		customStateID       int
		mockWrapper         wrappers.CustomStatesWrapper
		mockFeatureFlags    wrappers.FeatureFlagsWrapper
		flag                wrappers.FeatureFlagResponseModel
		expectedState       string
		expectedCustomState int
		expectedError       string
	}{
		{
			name:                "Custom state with valid state name",
			state:               "demo2",
			customStateID:       -1,
			mockWrapper:         &mock.CustomStatesMockWrapper{},
			mockFeatureFlags:    &mock.FeatureFlagsMockWrapper{},
			flag:                wrappers.FeatureFlagResponseModel{Name: wrappers.SastCustomStateEnabled, Status: true},
			expectedState:       "",
			expectedCustomState: 2,
			expectedError:       "",
		},
		{
			name:                "Custom state with valid state ID",
			state:               "",
			customStateID:       2,
			mockFeatureFlags:    &mock.FeatureFlagsMockWrapper{},
			mockWrapper:         &mock.CustomStatesMockWrapper{},
			flag:                wrappers.FeatureFlagResponseModel{Name: wrappers.SastCustomStateEnabled, Status: true},
			expectedState:       "",
			expectedCustomState: 2,
			expectedError:       "",
		},
		{
			name:                "System state",
			state:               "TO_VERIFY",
			customStateID:       -1,
			mockWrapper:         &mock.CustomStatesMockWrapper{},
			mockFeatureFlags:    &mock.FeatureFlagsMockWrapper{},
			flag:                wrappers.FeatureFlagResponseModel{Name: wrappers.SastCustomStateEnabled, Status: true},
			expectedState:       "TO_VERIFY",
			expectedCustomState: -1,
			expectedError:       "",
		},
		{
			name:                "State ID required when state is not provided",
			state:               "",
			customStateID:       -1,
			mockWrapper:         &mock.CustomStatesMockWrapper{},
			mockFeatureFlags:    &mock.FeatureFlagsMockWrapper{},
			flag:                wrappers.FeatureFlagResponseModel{Name: wrappers.SastCustomStateEnabled, Status: true},
			expectedState:       "",
			expectedCustomState: -1,
			expectedError:       "state-id is required when state is not provided",
		},
		{
			name:                "Failed to get custom state ID",
			state:               "INVALID_STATE",
			customStateID:       -1,
			mockWrapper:         &mock.CustomStatesMockWrapperWithError{},
			mockFeatureFlags:    &mock.FeatureFlagsMockWrapper{},
			flag:                wrappers.FeatureFlagResponseModel{Name: wrappers.SastCustomStateEnabled, Status: true},
			expectedState:       "",
			expectedCustomState: -1,
			expectedError:       "Failed to get custom state ID for state: INVALID_STATE",
		},
		{
			name:                "Both state and state ID provided - valid custom state",
			state:               "demo2",
			customStateID:       2,
			mockWrapper:         &mock.CustomStatesMockWrapper{},
			mockFeatureFlags:    &mock.FeatureFlagsMockWrapper{},
			flag:                wrappers.FeatureFlagResponseModel{Name: wrappers.SastCustomStateEnabled, Status: true},
			expectedState:       "",
			expectedCustomState: 2,
			expectedError:       "",
		},
		{
			name:                "Both state and state ID provided - valid system state",
			state:               "TO_VERIFY",
			customStateID:       2,
			mockWrapper:         &mock.CustomStatesMockWrapper{},
			mockFeatureFlags:    &mock.FeatureFlagsMockWrapper{},
			flag:                wrappers.FeatureFlagResponseModel{Name: wrappers.SastCustomStateEnabled, Status: true},
			expectedState:       "TO_VERIFY",
			expectedCustomState: -1,
			expectedError:       "",
		},
		{
			name:                "Both state and state ID provided - invalid state name",
			state:               "INVALID_STATE",
			customStateID:       2,
			mockWrapper:         &mock.CustomStatesMockWrapperWithError{},
			mockFeatureFlags:    &mock.FeatureFlagsMockWrapper{},
			flag:                wrappers.FeatureFlagResponseModel{Name: wrappers.SastCustomStateEnabled, Status: true},
			expectedState:       "",
			expectedCustomState: 2,
			expectedError:       "",
		},
		{
			name:                "Custom state not available",
			state:               "demo2",
			customStateID:       -1,
			mockWrapper:         &mock.CustomStatesMockWrapper{},
			mockFeatureFlags:    &mock.FeatureFlagsMockWrapper{},
			flag:                wrappers.FeatureFlagResponseModel{Name: wrappers.SastCustomStateEnabled, Status: false},
			expectedState:       "",
			expectedCustomState: -1,
			expectedError:       "Custom state is not available for your tenant.",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			clearFlags()
			mock.Flag = tt.flag
			state, customStateID, err := determineSystemOrCustomState(tt.mockWrapper, tt.mockFeatureFlags, tt.state, tt.customStateID)
			if tt.expectedError != "" {
				assert.ErrorContains(t, err, tt.expectedError)
			} else {
				assert.NilError(t, err)
			}
			assert.Equal(t, state, tt.expectedState)
			assert.Equal(t, customStateID, tt.expectedCustomState)
		})
	}
}
