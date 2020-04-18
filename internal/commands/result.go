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
			err = outputResults(cmd, resultResponseModel)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func outputResults(cmd *cobra.Command, model []wrappers.ResultResponseModel) error {
	if IsJSONFormat() {
		var resultsJSON []byte
		resultsJSON, err := json.Marshal(model)
		if err != nil {
			return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingAll)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(resultsJSON))
	} else if IsPrettyFormat() {
		for i := 0; i < len(model); i++ {
			outputSingleResult(&wrappers.ResultResponseModel{
				QueryID:      model[i].QueryID,
				QueryName:    model[i].QueryName,
				Severity:     model[i].Severity,
				CweID:        model[i].CweID,
				SimilarityID: model[i].SimilarityID,
				ID:           model[i].ID,
				FirstScanID:  model[i].FirstScanID,
				FirstFoundAt: model[i].FirstFoundAt,
				FoundAt:      model[i].FoundAt,
				Status:       model[i].Status,
			})
		}
	}
	return nil
}

func outputSingleResult(model *wrappers.ResultResponseModel) {
	fmt.Println("----------------------------")
	fmt.Println("Query ID:", model.QueryID)
	fmt.Println("Query Name:", model.QueryName)
	fmt.Println("Severity:", model.Severity)
	fmt.Println("CWE ID:", model.CweID)
	fmt.Println("Similarity ID:", model.SimilarityID)
}
