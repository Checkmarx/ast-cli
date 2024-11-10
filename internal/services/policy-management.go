package services

import (
	"slices"

	"github.com/checkmarx/ast-cli/internal/commands/policymanagement"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var noPolicyEvaluatingIDEs = []string{commonParams.EclipseAgent, commonParams.JetbrainsAgent, commonParams.VSCodeAgent, commonParams.VisualStudioAgent}

func HandlePolicyEvaluation(cmd *cobra.Command, policyWrapper wrappers.PolicyWrapper, scan *wrappers.ScanResponseModel) (*wrappers.PolicyResponseModel, error) {
	policyResponseModel := &wrappers.PolicyResponseModel{}
	policyOverrideFlag, _ := cmd.Flags().GetBool(commonParams.IgnorePolicyFlag)
	waitDelay, _ := cmd.Flags().GetInt(commonParams.WaitDelayFlag)
	agent := getAgent(cmd)

	if policyOverrideFlag || slices.Contains(noPolicyEvaluatingIDEs, agent) {
		logger.PrintIfVerbose("Skipping policy evaluation")
		return policyResponseModel, nil
	}

	policyTimeout, _ := cmd.Flags().GetInt(commonParams.PolicyTimeoutFlag)
	if policyTimeout < 0 {
		return nil, errors.Errorf("--%s should be equal or higher than 0", commonParams.PolicyTimeoutFlag)
	}

	return policymanagement.HandlePolicyWait(waitDelay, policyTimeout, policyWrapper, scan.ID, scan.ProjectID, cmd)
}

func getAgent(cmd *cobra.Command) string {
	agent, _ := cmd.Flags().GetString(commonParams.AgentFlag)
	if agent == "" {
		return commonParams.DefaultAgent
	}
	return agent
}
