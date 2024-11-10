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

func HandlePolicyEvaluation(cmd *cobra.Command, policyWrapper wrappers.PolicyWrapper, scan *wrappers.ScanResponseModel, ignorePolicy bool, agent string, waitDelay, policyTimeout int) (*wrappers.PolicyResponseModel, error) {
	policyResponseModel := &wrappers.PolicyResponseModel{}

	if ignorePolicy || slices.Contains(noPolicyEvaluatingIDEs, agent) {
		logger.PrintIfVerbose("Skipping policy evaluation")
		return policyResponseModel, nil
	}

	if policyTimeout < 0 {
		return nil, errors.Errorf("--%s should be equal or higher than 0", commonParams.PolicyTimeoutFlag)
	}

	return policymanagement.HandlePolicyWait(waitDelay, policyTimeout, policyWrapper, scan.ID, scan.ProjectID, cmd)
}
