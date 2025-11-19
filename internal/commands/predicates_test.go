//go:build !integration

package commands

import (
	"fmt"
	"testing"
	"time"

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
	mockResultsPredicatesWrapper := &mock.ResultsPredicatesWrapper{}
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
	mockResultsPredicatesWrapper := &mock.ResultsPredicatesWrapper{}
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
	mockResultsPredicatesWrapper := &mock.ResultsPredicatesWrapper{}
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

func TestPrepareScaTriagePayload(t *testing.T) {
	tests := []struct {
		name                 string
		vulnerabilityDetails []string
		comment              string
		state                string
		projectId            string
		expectedError        string
	}{
		{
			name: "Missing packageName",
			vulnerabilityDetails: []string{
				"packageVersion=4.17.20",
				"packageManager=npm",
				"vulnerabilityId=CVE-2021-23337",
			},
			comment:       "Testing missing package name",
			state:         "NOT_EXPLOITABLE",
			projectId:     "test-project-123",
			expectedError: "Package name is required",
		},
		{
			name: "Missing packageVersion",
			vulnerabilityDetails: []string{
				"packageName=lodash",
				"packageManager=npm",
				"vulnerabilityId=CVE-2021-23337",
			},
			comment:       "Testing missing package version",
			state:         "NOT_EXPLOITABLE",
			projectId:     "test-project-123",
			expectedError: "Package version is required",
		},
		{
			name: "Missing packageManager",
			vulnerabilityDetails: []string{
				"packageName=lodash",
				"packageVersion=4.17.20",
				"vulnerabilityId=CVE-2021-23337",
			},
			comment:       "Testing missing package manager",
			state:         "NOT_EXPLOITABLE",
			projectId:     "test-project-123",
			expectedError: "Package manager is required",
		},
		{
			name: "Invalid vulnerability format - no equals sign",
			vulnerabilityDetails: []string{
				"packageNamelodash",
				"packageVersion=4.17.20",
				"packageManager=npm",
			},
			comment:       "Testing invalid format",
			state:         "NOT_EXPLOITABLE",
			projectId:     "test-project-123",
			expectedError: "Invalid vulnerabilities. It should be in a KEY=VALUE format",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			payload, err := prepareScaTriagePayload(tt.vulnerabilityDetails, tt.comment, tt.state, tt.projectId)
			if tt.expectedError != "" {
				assert.ErrorContains(t, err, tt.expectedError)
			} else {
				assert.NilError(t, err)
				assert.Assert(t, payload != nil, "Expected payload to be non-nil")
			}
		})
	}
}

func TestPrepareScaTriagePayloadWithMissingVulnerabilities(t *testing.T) {
	payload, err := prepareScaTriagePayload(nil, "Testing missing vulnerabilities", "NOT_EXPLOITABLE", "test-project-123")
	assert.ErrorContains(t, err, "Vulnerabilities details are required.")
	assert.Assert(t, payload == nil, "Expected payload to be nil")
}

func TestRunShowTriageCommandForSCAWithMissingVulnerabilities(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		"triage",
		"show",
		"--project-id",
		"MOCK",
		"--scan-type",
		"sca",
	)
	// SCA triage show requires vulnerabilities flag
	assert.Assert(t, err != nil, "Expected error when vulnerabilities flag is missing")
}

func TestRunShowTriageCommandForSCAWithMultipleProjects(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		"triage",
		"show",
		"--project-id",
		"MOCK1,MOCK2",
		"--scan-type",
		"sca",
		"--vulnerabilities",
		"packageName=lodash,packageVersion=4.17.20,packageManager=npm",
	)
	assert.ErrorContains(t, err, "Multiple project-ids are not allowed")
}

func TestToScaPredicateResultView(t *testing.T) {
	// Arrange: Create sample SCA predicate result
	createdAt1, _ := time.Parse(time.RFC3339, "2024-01-15T10:00:00Z")
	createdAt2, _ := time.Parse(time.RFC3339, "2024-01-16T12:00:00Z")

	scaPredicateResult := wrappers.ScaPredicateResult{
		Context: wrappers.Context{
			VulnerabilityId: "CVE-2021-23337",
			PackageName:     "lodash",
			PackageVersion:  "4.17.20",
			PackageManager:  "npm",
		},
		Actions: []wrappers.Action{
			{
				ActionType:  "ChangeState",
				ActionValue: "NOT_EXPLOITABLE",
				Message:     "This is not exploitable in our context",
				UserName:    "test-user",
				CreatedAt:   createdAt1,
				Enabled:     true,
			},
			{
				ActionType:  "ChangeState",
				ActionValue: "CONFIRMED",
				Message:     "Actually, this needs to be fixed",
				UserName:    "test-user-2",
				CreatedAt:   createdAt2,
				Enabled:     true,
			},
		},
	}

	// Act: Call the toScaPredicateResultView function
	result := toScaPredicateResultView(scaPredicateResult)

	// Assert: Verify the conversion
	assert.Equal(t, len(result), 2, "Expected 2 predicate result views")

	// Check first action
	assert.Equal(t, result[1].VulnerabilityID, "CVE-2021-23337")
	assert.Equal(t, result[1].PackageName, "lodash")
	assert.Equal(t, result[1].PackageVersion, "4.17.20")
	assert.Equal(t, result[1].PackageManager, "npm")
	assert.Equal(t, result[1].State, "NOT_EXPLOITABLE")
	assert.Equal(t, result[1].Comment, "This is not exploitable in our context")
	assert.Equal(t, result[1].CreatedBy, "test-user")
	assert.Equal(t, result[1].CreatedAt, createdAt1)

	// Check second action
	assert.Equal(t, result[0].State, "CONFIRMED")
	assert.Equal(t, result[0].Comment, "Actually, this needs to be fixed")
	assert.Equal(t, result[0].CreatedBy, "test-user-2")
}

func TestToScaPredicateResultView_EmptyActions(t *testing.T) {
	// Arrange: Create SCA predicate result with no actions
	scaPredicateResult := wrappers.ScaPredicateResult{
		Context: wrappers.Context{
			VulnerabilityId: "CVE-2021-23337",
			PackageName:     "lodash",
			PackageVersion:  "4.17.20",
			PackageManager:  "npm",
		},
		Actions: []wrappers.Action{},
	}

	// Act: Call the toScaPredicateResultView function
	result := toScaPredicateResultView(scaPredicateResult)

	// Assert: Verify empty result
	assert.Equal(t, len(result), 0, "Expected empty predicate result views")
}
