package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"

	wrappers "github.com/checkmarxDev/ast-cli/internal/wrappers"
	scansRESTApi "github.com/checkmarxDev/scans/api/v1/rest/scans"

	"github.com/spf13/cobra"
)

const (
	failedCreating    = "Failed creating a scan"
	failedGetting     = "Failed getting a scan"
	failedGettingTags = "Failed getting tags"
	failedDeleting    = "Failed deleting a scan"
	failedGettingAll  = "Failed getting all"
)

func NewScanCommand(scansWrapper wrappers.ScansWrapper, uploadsWrapper wrappers.UploadsWrapper) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage AST scans",
	}

	createScanCmd := &cobra.Command{
		Use:   "create",
		Short: "Create and run a new scan",
		RunE:  runCreateScanCommand(scansWrapper, uploadsWrapper),
	}
	createScanCmd.PersistentFlags().StringP(sourcesFlag, sourcesFlagSh, "",
		"A path to the sources file to scan")
	createScanCmd.PersistentFlags().StringP(inputFlag, inputFlagSh, "",
		"The object representing the requested scan, in JSON format")
	createScanCmd.PersistentFlags().StringP(inputFileFlag, inputFileFlagSh, "",
		"A file holding the requested scan object in JSON format. Takes precedence over --input")

	listScansCmd := &cobra.Command{
		Use:   "list",
		Short: "List all scans in the system",
		RunE:  runGetAllScansCommand(scansWrapper),
	}
	listScansCmd.PersistentFlags().Uint64P(limitFlag, limitFlagSh, 0, limitUsage)
	listScansCmd.PersistentFlags().Uint64P(offsetFlag, offsetFlagSh, 0, offsetUsage)

	showScanCmd := &cobra.Command{
		Use:   "show",
		Short: "Show information about a scan",
		RunE:  runGetScanByIDCommand(scansWrapper),
	}

	deleteScanCmd := &cobra.Command{
		Use:   "delete",
		Short: "Stops a scan from running",
		RunE:  runDeleteScanCommand(scansWrapper),
	}

	tagsCmd := &cobra.Command{
		Use:   "tags",
		Short: "Get a list of all available tags to filter by",
		RunE:  runGetTagsCommand(scansWrapper),
	}

	scanCmd.AddCommand(createScanCmd, showScanCmd, listScansCmd, deleteScanCmd, tagsCmd)
	return scanCmd
}

func runCreateScanCommand(scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var input []byte
		var err error

		var scanInputFile string
		var scanInput string
		var sourcesFile string

		scanInput, _ = cmd.Flags().GetString(inputFlag)
		scanInputFile, _ = cmd.Flags().GetString(inputFileFlag)
		sourcesFile, _ = cmd.Flags().GetString(sourcesFlag)

		PrintIfVerbose(fmt.Sprintf("%s: %s", inputFlag, scanInput))
		PrintIfVerbose(fmt.Sprintf("%s: %s", inputFileFlag, scanInputFile))
		PrintIfVerbose(fmt.Sprintf("%s: %s", sourcesFlag, sourcesFile))

		if scanInputFile != "" {
			// Reading from input file
			PrintIfVerbose(fmt.Sprintf("Reading input from file %s", scanInputFile))
			input, err = ioutil.ReadFile(scanInputFile)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to open input file", failedCreating)
			}
		} else if scanInput != "" {
			// Reading from standard input
			PrintIfVerbose("Reading input from console")
			input = bytes.NewBufferString(scanInput).Bytes()
		} else {
			// No input was given
			return errors.Errorf("%s: no input was given\n", failedCreating)
		}
		var scanModel = scansRESTApi.Scan{}
		var scanResponseModel *scansRESTApi.ScanResponseModel
		var errorModel *scansRESTApi.ErrorModel
		// Try to parse to a scan model in order to manipulate the request payload
		err = json.Unmarshal(input, &scanModel)
		if err != nil {
			return errors.Wrapf(err, "%s: Input in bad format", failedCreating)
		}
		if sourcesFile != "" {
			// Send a request to uploads service
			var preSignedURL *string
			preSignedURL, err = uploadsWrapper.UploadFile(sourcesFile)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to upload sources file\n", failedCreating)
			}
			// We are in upload mode - populate fields accordingly
			projectHandlerModel := scansRESTApi.UploadProjectHandler{
				URL: *preSignedURL,
			}
			var projectHandlerModelSerialized []byte
			projectHandlerModelSerialized, err = json.Marshal(projectHandlerModel)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to upload sources file: Failed to serialize project handler",
					failedCreating)
			}
			scanModel.Project.Type = scansRESTApi.UploadProject
			scanModel.Project.Handler = projectHandlerModelSerialized
		}
		var payload []byte
		payload, _ = json.Marshal(scanModel)
		PrintIfVerbose(fmt.Sprintf("Payload to scans service: %s\n", string(payload)))

		scanResponseModel, errorModel, err = scansWrapper.Create(&scanModel)
		if err != nil {
			return errors.Wrapf(err, "%s", failedCreating)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedCreating, errorModel.Code, errorModel.Message)
		} else if scanResponseModel != nil {
			var responseModelJSON []byte
			responseModelJSON, err = json.Marshal(scanResponseModel)
			if err != nil {
				return errors.Wrapf(err, "%s: failed to serialize scan response ", failedCreating)
			}
			cmdOut := cmd.OutOrStdout()
			fmt.Fprintln(os.Stdout, "Scan created successfully")
			fmt.Fprintln(cmdOut, string(responseModelJSON))
		}
		return nil
	}
}

func runGetAllScansCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var allScansModel *scansRESTApi.SlicedScansResponseModel
		var errorModel *scansRESTApi.ErrorModel
		var err error
		limit, offset := getLimitAndOffset(cmd)

		allScansModel, errorModel, err = scansWrapper.Get(limit, offset)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingAll)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedGettingAll, errorModel.Code, errorModel.Message)
		} else if allScansModel != nil && allScansModel.Scans != nil {
			cmdOut := cmd.OutOrStdout()
			if cmdOut != os.Stdout {
				var allScansJSON []byte
				allScansJSON, err = json.Marshal(allScansModel)
				if err != nil {
					return errors.Wrapf(err, "%s: failed to serialize scan response ", failedGettingAll)
				}
				fmt.Fprintln(cmdOut, string(allScansJSON))
			}
			for _, scan := range allScansModel.Scans {
				var responseModelJSON []byte
				responseModelJSON, err = json.Marshal(scan)
				if err != nil {
					return errors.Wrapf(err, "%s: failed to serialize project response ", failedGettingAll)
				}
				fmt.Fprintln(os.Stdout, "----------------------------")
				fmt.Fprintln(os.Stdout, string(responseModelJSON))
			}
			fmt.Fprintln(os.Stdout, "----------------------------")
		}
		return nil
	}
}

func runGetScanByIDCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var scanResponseModel *scansRESTApi.ScanResponseModel
		var errorModel *scansRESTApi.ErrorModel
		var err error
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a scan ID", failedGetting)
		}
		scanID := args[0]
		scanResponseModel, errorModel, err = scansWrapper.GetByID(scanID)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGetting)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGetting, errorModel.Code, errorModel.Message)
		} else if scanResponseModel != nil {
			var responseModelJSON []byte
			responseModelJSON, err = json.Marshal(scanResponseModel)
			if err != nil {
				return errors.Wrapf(err, "%s: failed to serialize scan response ", failedGetting)
			}
			cmdOut := cmd.OutOrStdout()
			fmt.Fprintf(os.Stdout, "-----Scan ID %s-----\n", scanResponseModel.ID)
			fmt.Fprintln(cmdOut, string(responseModelJSON))
		}
		return nil
	}
}

func runDeleteScanCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var errorModel *scansRESTApi.ErrorModel
		var err error
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a scan ID", failedDeleting)
		}
		scanID := args[0]
		errorModel, err = scansWrapper.Delete(scanID)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedDeleting)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedDeleting, errorModel.Code, errorModel.Message)
		}
		return nil
	}
}

func runGetTagsCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var tags *[]string
		var errorModel *scansRESTApi.ErrorModel
		var err error
		tags, errorModel, err = scansWrapper.Tags()
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingTags)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingTags, errorModel.Code, errorModel.Message)
		} else if tags != nil {
			var tagsJSON []byte
			tagsJSON, err = json.Marshal(tags)
			if err != nil {
				return errors.Wrapf(err, "%s: failed to serialize scan tags response ", failedGettingTags)
			}
			cmdOut := cmd.OutOrStdout()
			fmt.Fprintln(os.Stdout, "-----Tags-----")
			fmt.Fprintln(cmdOut, string(tagsJSON))
		}
		return nil
	}
}
