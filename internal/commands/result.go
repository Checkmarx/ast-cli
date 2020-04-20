package commands

import (
	"encoding/json"
	"fmt"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"

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
	listResultsCmd.PersistentFlags().StringSlice(filterFlag, []string{}, filterScanListFlagUsage)

	resultCmd.AddCommand(listResultsCmd)
	return resultCmd
}

func runGetResultByScanIDCommand(resultsWrapper wrappers.ResultsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var resultResponseModel *wrappers.ResultsResponseModel
		var errorModel *wrappers.ErrorModel
		var err error
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a scan ID", failedListingResults)
		}

		scanID := args[0]
		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingResults)
		}
		params[commonParams.ScanIDQueryParam] = scanID

		resultResponseModel, errorModel, err = resultsWrapper.GetByScanID(params)
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingResults)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedListingResults, errorModel.Code, errorModel.Message)
		} else if resultResponseModel != nil {
			if IsJSONFormat() {
				var resultsJSON []byte
				resultsJSON, err = json.Marshal(resultResponseModel)
				if err != nil {
					return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingAll)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(resultsJSON))
			} else if IsPrettyFormat() {
				err = outputResultsPretty(resultResponseModel.Results)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func outputResultsPretty(results []wrappers.ResultResponseModel) error {
	fmt.Println("************ Results ************")
	for i := 0; i < len(results); i++ {
		outputSingleResult(&wrappers.ResultResponseModel{
			QueryID:                         results[i].QueryID,
			QueryName:                       results[i].QueryName,
			Severity:                        results[i].Severity,
			CweID:                           results[i].CweID,
			SimilarityID:                    results[i].SimilarityID,
			UniqueID:                        results[i].UniqueID,
			FirstScanID:                     results[i].FirstScanID,
			FirstFoundAt:                    results[i].FirstFoundAt,
			FoundAt:                         results[i].FoundAt,
			Status:                          results[i].Status,
			PathSystemID:                    results[i].PathSystemID,
			PathSystemIDBySimiAndFilesPaths: results[i].PathSystemIDBySimiAndFilesPaths,
			Nodes:                           results[i].Nodes,
		})
		fmt.Println()
	}
	return nil
}

func outputSingleResult(model *wrappers.ResultResponseModel) {
	fmt.Println("Result Unique ID:", model.UniqueID)
	fmt.Println("Query ID:", model.QueryID)
	fmt.Println("Query Name:", model.QueryName)
	fmt.Println("Severity:", model.Severity)
	fmt.Println("CWE ID:", model.CweID)
	fmt.Println("Similarity ID:", model.SimilarityID)
	fmt.Println("First Scan ID:", model.FirstScanID)
	fmt.Println("Found At:", model.FoundAt)
	fmt.Println("First Found At:", model.FirstFoundAt)
	fmt.Println("Status:", model.Status)
	fmt.Println("Path System ID:", model.PathSystemID)
	fmt.Println("Path System ID (by similarity and file paths):", model.PathSystemIDBySimiAndFilesPaths)
	fmt.Println()
	fmt.Println("************ Nodes ************")
	for i := 0; i < len(model.Nodes); i++ {
		outputSingleResultNodePretty(model.Nodes[i])
		fmt.Println()
	}
}
