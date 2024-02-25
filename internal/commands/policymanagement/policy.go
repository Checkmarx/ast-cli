package policymanagement

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	failedGetting      = "Failed showing a scan"
	maxPollingWaitTime = 60
	completedPolicy    = "COMPLETED"
	nonePolicy         = "NONE"
	evaluatingPolicy   = "EVALUATING"
)

func HandlePolicyWait(
	waitDelay,
	timeoutMinutes int,
	policyWrapper wrappers.PolicyWrapper,
	scanID,
	projectID string,
	cmd *cobra.Command,
) (*wrappers.PolicyResponseModel, error) {
	policyResponseModel, err := waitForPolicyCompletion(
		waitDelay,
		timeoutMinutes,
		policyWrapper,
		scanID,
		projectID,
		cmd)
	if err != nil {
		verboseFlag, _ := cmd.Flags().GetBool(commonParams.DebugFlag)
		if verboseFlag {
			logger.PrintIfVerbose("Policy evaluation failed")
		}
		return nil, err
	}
	return policyResponseModel, nil
}

func waitForPolicyCompletion(
	waitDelay int,
	timeoutMinutes int,
	policyWrapper wrappers.PolicyWrapper,
	scanID,
	projectID string,
	cmd *cobra.Command,
) (*wrappers.PolicyResponseModel, error) {
	logger.PrintIfVerbose("Waiting for policy evaluation to complete for scanID:" + scanID + " and projectID:" + projectID)
	var policyResponseModel *wrappers.PolicyResponseModel
	timeout := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)
	fixedWait := time.Duration(waitDelay) * time.Second
	i := uint64(0)
	if !cmd.Flags().Changed(commonParams.RetryDelayFlag) {
		viper.Set(commonParams.RetryDelayFlag, commonParams.RetryDelayPollingDefault)
	}
	for {
		variableWait := time.Duration(math.Min(float64(i/uint64(waitDelay)), maxPollingWaitTime)) * time.Second
		waitDuration := fixedWait + variableWait
		logger.PrintfIfVerbose("Sleeping %v before polling", waitDuration)
		time.Sleep(waitDuration)
		evaluated := false
		var err error
		evaluated, policyResponseModel, err = isPolicyEvaluated(policyWrapper, scanID, projectID)
		if err != nil {
			return nil, err
		}
		if evaluated {
			break
		}
		if timeoutMinutes > 0 && time.Now().After(timeout) {
			logger.PrintfIfVerbose("Timeout of %d minute(s) for policy evaluation reached", timeoutMinutes)
			return nil, nil
		}
		i++
	}
	logger.PrintIfVerbose("Policy evaluation completed with status" + policyResponseModel.Status)
	return policyResponseModel, nil
}

func isPolicyEvaluated(
	policyWrapper wrappers.PolicyWrapper,
	scanID,
	projectID string,
) (bool, *wrappers.PolicyResponseModel, error) {
	var errorModel *wrappers.WebError
	var err error
	var policyResponseModel *wrappers.PolicyResponseModel
	var params = make(map[string]string)

	params["scanId"] = scanID
	params["astProjectId"] = projectID

	policyResponseModel, errorModel, err = policyWrapper.EvaluatePolicy(params)
	if err != nil {
		return false, nil, err
	}
	if errorModel != nil {
		log.Fatalf(fmt.Sprintf("%s: CODE: %d, %s", failedGetting, errorModel.Code, errorModel.Message))
	} else if policyResponseModel != nil {
		if policyResponseModel.Status == evaluatingPolicy {
			log.Println("Policy status: ", policyResponseModel.Status)
			return false, nil, nil
		}
	}
	// Case the policy is evaluated or None
	logger.PrintIfVerbose("Policy evaluation finished with status: " + policyResponseModel.Status)
	if policyResponseModel.Status == completedPolicy || policyResponseModel.Status == nonePolicy {
		logger.PrintIfVerbose("Policy status: " + policyResponseModel.Status)
		return true, policyResponseModel, nil
	}
	return true, nil, nil
}
