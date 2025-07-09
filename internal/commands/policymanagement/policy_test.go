package policymanagement

//Checking the tests

import (
	"errors"
	"testing"
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

type mockPolicyWrapper struct{}

func (m *mockPolicyWrapper) EvaluatePolicy(params map[string]string) (*wrappers.PolicyResponseModel, *wrappers.WebError, error) {
	return &wrappers.PolicyResponseModel{
		Status: completedPolicy,
	}, nil, nil
}

func TestHandlePolicyWait_Success(t *testing.T) {
	mockWrapper := &mockPolicyWrapper{}
	cmd := &cobra.Command{}

	response, err := HandlePolicyWait(1, 5, mockWrapper, "scanID", "projectID", cmd)
	assert.NilError(t, err, "Expected no error, got %v", err)
	assert.Equal(t, response.Status, completedPolicy, "Expected status %s, got %s", completedPolicy, response.Status)
}

type mockFailingPolicyWrapper struct{}

func (m *mockFailingPolicyWrapper) EvaluatePolicy(params map[string]string) (*wrappers.PolicyResponseModel, *wrappers.WebError, error) {
	return nil, nil, errors.New("mock error")
}

func TestHandlePolicyWait_Error(t *testing.T) {
	mockWrapper := &mockFailingPolicyWrapper{}
	cmd := &cobra.Command{}

	_, err := HandlePolicyWait(1, 5, mockWrapper, "scanID", "projectID", cmd)
	assert.ErrorContains(t, err, "mock error", "Expected error, got %v", err)
}

func TestWaitForPolicyCompletion_Timeout(t *testing.T) {
	mockWrapper := &mockPolicyWrapper{}
	cmd := &cobra.Command{}

	start := time.Now()
	_, err := waitForPolicyCompletion(1, 1, mockWrapper, "scanID", "projectID", cmd)
	duration := time.Since(start)
	assert.NilError(t, err, "Expected no error, got %v", err)

	assert.Assert(t, duration.Seconds() >= 1, "Expected timeout duration of at least 1 second, got %v", duration)
}
