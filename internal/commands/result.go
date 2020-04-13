package commands

import (
	"encoding/json"
	"fmt"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedListingResults = "Failed listing results"
)

func NewResultCommand(resultsWrapper wrappers.ResultsWrapper) *cobra.Command {
	resultCmd := &cobra.Command{
		Use:   "result",
		Short: "Retrieve AST results",
	}

	listResultsCmd := &cobra.Command{
		Use:   "list",
		Short: "List results for a given scan",
		RunE:  runGetResultByScanIDCommand(resultsWrapper),
	}
	listResultsCmd.PersistentFlags().Uint64P(limitFlag, limitFlagSh, 0, limitUsage)
	listResultsCmd.PersistentFlags().Uint64P(offsetFlag, offsetFlagSh, 0, offsetUsage)

	resultCmd.AddCommand(listResultsCmd)
	return resultCmd
}

func runGetResultByScanIDCommand(resultsWrapper wrappers.ResultsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var resultResponseModel []wrappers.ResultResponseModel
		var errorModel *wrappers.ResultError
		var err error
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a scan ID", failedListingResults)
		}
		scanID := args[0]
		limit, offset := getLimitAndOffset(cmd)

		resultResponseModel, errorModel, err = resultsWrapper.GetByScanID(scanID, limit, offset)
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingResults)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedListingResults, errorModel.Code, errorModel.Message)
		} else if resultResponseModel != nil {
			var responseModelJSON []byte
			responseModelJSON, err = json.Marshal(resultResponseModel)
			if err != nil {
				return errors.Wrapf(err, "%s: failed to serialize results response ", failedListingResults)
			}
			cmdOut := cmd.OutOrStdout()
			fmt.Fprintln(cmdOut, string(responseModelJSON))
		}
		return nil
	}
}
