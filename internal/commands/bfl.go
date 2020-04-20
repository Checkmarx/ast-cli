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
	failedGettingBfl = "Failed getting BFL"
)

func NewBFLCommand(bflWrapper wrappers.BFLWrapper) *cobra.Command {
	bflCmd := &cobra.Command{
		Use:   "bfl",
		Short: "Retrieve Best Fix Location for a given scan ID",
		RunE:  runGetBFLByScanIDCommand(bflWrapper),
	}
	bflCmd.PersistentFlags().StringSlice(filterFlag, []string{}, filterScanListFlagUsage)

	return bflCmd
}

func runGetBFLByScanIDCommand(bflWrapper wrappers.BFLWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var bflResponseModel *wrappers.BFLResponseModel
		var errorModel *wrappers.ErrorModel

		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a scan ID", failedGettingBfl)
		}

		scanID := args[0]
		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingBfl)
		}
		params[commonParams.ScanIDQueryParam] = scanID

		bflResponseModel, errorModel, err = bflWrapper.GetByScanID(params)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingBfl)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingBfl, errorModel.Code, errorModel.Message)
		} else if bflResponseModel != nil {
			err = outputBFL(cmd, bflResponseModel)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func outputBFL(cmd *cobra.Command, model *wrappers.BFLResponseModel) error {
	if IsJSONFormat() {
		var bflJSON []byte
		bflJSON, err := json.Marshal(model)
		if err != nil {
			return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingBfl)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(bflJSON))
	} else if IsPrettyFormat() {
		fmt.Println("************ Best Fix Location ************")
		fmt.Println("BFL ID:", model.ID)
		fmt.Println()
		for i := 0; i < len(model.Trees); i++ {
			fmt.Println("************ Tree ************")
			fmt.Println("ID:", model.Trees[i].ID)
			fmt.Println()
			fmt.Println("************ BFL Node ************")
			bfl := model.Trees[i].BFL
			outputSingleResultNodePretty(&wrappers.ResultNode{
				Column:       bfl.Column,
				FileName:     bfl.FileName,
				FullName:     bfl.FullName,
				Length:       bfl.Length,
				Line:         bfl.Line,
				MethodLine:   bfl.MethodLine,
				Name:         bfl.Name,
				NodeID:       bfl.NodeID,
				DomType:      bfl.DomType,
				NodeSystemID: bfl.NodeSystemID,
			})
			fmt.Println()
			err := outputResultsPretty(model.Trees[i].Results)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func outputSingleResultNodePretty(model *wrappers.ResultNode) {
	fmt.Println("Name:", model.Name)
	fmt.Println("File Name:", model.FileName)
	fmt.Println("Full Name:", model.FullName)
	fmt.Println("Length:", model.Length)
	fmt.Println("Column:", model.Column)
	fmt.Println("Line:", model.Line)
	fmt.Println("Method Line:", model.MethodLine)
	fmt.Println("Node System ID:", model.NodeSystemID)
}
