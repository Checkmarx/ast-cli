package policymanagement

import (
	"errors"
	"testing"
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
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

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Error("Expected response, got nil")
	}

	if response.Status != completedPolicy {
		t.Errorf("Expected status %s, got %s", completedPolicy, response.Status)
	}
}

type mockFailingPolicyWrapper struct{}

func (m *mockFailingPolicyWrapper) EvaluatePolicy(params map[string]string) (*wrappers.PolicyResponseModel, *wrappers.WebError, error) {
	return nil, nil, errors.New("mock error")
}

func TestHandlePolicyWait_Error(t *testing.T) {
	mockWrapper := &mockFailingPolicyWrapper{}
	cmd := &cobra.Command{}

	_, err := HandlePolicyWait(1, 5, mockWrapper, "scanID", "projectID", cmd)

	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestWaitForPolicyCompletion_Timeout(t *testing.T) {
	mockWrapper := &mockPolicyWrapper{}
	cmd := &cobra.Command{}

	start := time.Now()
	_, err := waitForPolicyCompletion(1, 1, mockWrapper, "scanID", "projectID", cmd)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if duration.Seconds() < 1 {
		t.Errorf("Expected timeout duration of at least 1 second, got %v", duration)
	}
}
