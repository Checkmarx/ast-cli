package services

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

var (
	scanWrapper       = mock.ScansMockWrapper{}
	scanResults, _, _ = scanWrapper.GetByID("1234")
	policyWrapper     = &mock.PolicyMockWrapper{}
	policyModel, _, _ = policyWrapper.EvaluatePolicy(map[string]string{})
)

func TestHandlePolicyEvaluation(t *testing.T) {
	commonArgs := args{
		cmd:           &cobra.Command{},
		policyWrapper: policyWrapper,
		scan:          scanResults,
		ignorePolicy:  false,
		waitDelay:     1,
		policyTimeout: 1,
	}

	testCases := []struct {
		name        string
		args        args
		expected    *wrappers.PolicyResponseModel
		expectError bool
	}{
		{
			name: "DefaultAgent_Success",
			args: args{
				cmd:           commonArgs.cmd,
				policyWrapper: commonArgs.policyWrapper,
				scan:          commonArgs.scan,
				ignorePolicy:  commonArgs.ignorePolicy,
				agent:         params.DefaultAgent,
				waitDelay:     commonArgs.waitDelay,
				policyTimeout: commonArgs.policyTimeout,
			},
			expected:    policyModel,
			expectError: false,
		},
		{
			name: "EclipseAgent_NoPolicyEvaluation",
			args: args{
				cmd:           commonArgs.cmd,
				policyWrapper: commonArgs.policyWrapper,
				scan:          commonArgs.scan,
				ignorePolicy:  commonArgs.ignorePolicy,
				agent:         params.EclipseAgent,
				waitDelay:     commonArgs.waitDelay,
				policyTimeout: commonArgs.policyTimeout,
			},
			expected:    &wrappers.PolicyResponseModel{},
			expectError: false,
		},
		{
			name: "VSCodeAgent_NoPolicyEvaluation",
			args: args{
				cmd:           commonArgs.cmd,
				policyWrapper: commonArgs.policyWrapper,
				scan:          commonArgs.scan,
				ignorePolicy:  commonArgs.ignorePolicy,
				agent:         params.VSCodeAgent,
				waitDelay:     commonArgs.waitDelay,
				policyTimeout: commonArgs.policyTimeout,
			},
			expected:    &wrappers.PolicyResponseModel{},
			expectError: false,
		},
		{
			name: "VisualStudioAgent_NoPolicyEvaluation",
			args: args{
				cmd:           commonArgs.cmd,
				policyWrapper: commonArgs.policyWrapper,
				scan:          commonArgs.scan,
				ignorePolicy:  commonArgs.ignorePolicy,
				agent:         params.VisualStudioAgent,
				waitDelay:     commonArgs.waitDelay,
				policyTimeout: commonArgs.policyTimeout,
			},
			expected:    &wrappers.PolicyResponseModel{},
			expectError: false,
		},
		{
			name: "JetbrainsAgent_NoPolicyEvaluation",
			args: args{
				cmd:           commonArgs.cmd,
				policyWrapper: commonArgs.policyWrapper,
				scan:          commonArgs.scan,
				ignorePolicy:  commonArgs.ignorePolicy,
				agent:         params.JetbrainsAgent,
				waitDelay:     commonArgs.waitDelay,
				policyTimeout: commonArgs.policyTimeout,
			},
			expected:    &wrappers.PolicyResponseModel{},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result, err := HandlePolicyEvaluation(
				tc.args.cmd,
				tc.args.policyWrapper,
				tc.args.scan,
				tc.args.ignorePolicy,
				tc.args.asyncFlag,
				tc.args.agent,
				tc.args.waitDelay,
				tc.args.policyTimeout,
			)
			assert.Equal(t, tc.expected, result)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type args struct {
	cmd           *cobra.Command
	policyWrapper wrappers.PolicyWrapper
	scan          *wrappers.ScanResponseModel
	ignorePolicy  bool
	asyncFlag     bool
	agent         string
	waitDelay     int
	policyTimeout int
}
