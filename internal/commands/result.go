package commands

import (
	"encoding/json"
	"fmt"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedGettingResults = "Failed getting results"
)

func NewResultCommand(resultsWrapper wrappers.ResultsWrapper) *cobra.Command {
	resultCmd := &cobra.Command{
		Use:   "result",
		Short: "Retrieve AST results",
	}

	getResultsCmd := &cobra.Command{
		Use:   "get",
		Short: "Returns results for a given scan",
		RunE:  runGetResultByScanIDCommand(resultsWrapper),
	}
	getResultsCmd.PersistentFlags().Uint64P(limitFlag, limitFlagSh, 0, limitUsage)
	getResultsCmd.PersistentFlags().Uint64P(offsetFlag, offsetFlagSh, 0, offsetUsage)

	resultCmd.AddCommand(getResultsCmd)
	return resultCmd
}

func runGetResultByScanIDCommand(resultsWrapper wrappers.ResultsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var resultResponseModel []wrappers.ResultResponseModel
		var errorModel *wrappers.ResultError
		var err error
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a scan ID", failedGettingResults)
		}
		scanID := args[0]
		limit, offset := getLimitAndOffset(cmd)

		resultResponseModel, errorModel, err = resultsWrapper.GetByScanID(scanID, limit, offset)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingResults)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingResults, errorModel.Code, errorModel.Message)
		} else if resultResponseModel != nil {
			var responseModelJSON []byte
			responseModelJSON, err = json.Marshal(resultResponseModel)
			if err != nil {
				return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingResults)
			}
			cmdOut := cmd.OutOrStdout()
			fmt.Fprintln(cmdOut, string(responseModelJSON))
		}
		return nil
	}
}
