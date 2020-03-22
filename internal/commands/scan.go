package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/pkg/errors"

	wrappers "github.com/checkmarxDev/ast-cli/internal/wrappers"
	scansApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
	"github.com/checkmarxDev/scans/pkg/scans"
	"github.com/spf13/cobra"
)

// TODO print flags if verbose

const (
	failedCreating = "Failed creating a scan"
	failedGetting  = "Failed getting a scan"
)

func NewScanCommand(scansWrapper wrappers.ScansWrapper, uploadsWrapper wrappers.UploadsWrapper) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage AST scans",
	}

	createScanCmd := &cobra.Command{
		Use:   "create",
		Short: "Creates and runs a new scan",
		RunE:  runCreateScanCommand(scansWrapper, uploadsWrapper),
	}
	createScanCmd.PersistentFlags().StringP(sourcesFlag, sourcesFlagSh, "",
		"A path to the sources file to scan")
	createScanCmd.PersistentFlags().StringP(inputFlag, inputFlagSh, "",
		"The object representing the requested scan, in JSON format")
	createScanCmd.PersistentFlags().StringP(inputFileFlag, inputFileFlagSh, "",
		"A file holding the requested scan object in JSON format. Takes precedence over --input")
	createScanCmd.PersistentFlags().Bool(incrementalFlag, false,
		"Make this scan incremental ")

	getAllScanCmd := &cobra.Command{
		Use:   "get-all",
		Short: "Returns all scans in the system",
		RunE:  runGetAllScansCommand(scansWrapper),
	}

	getScanCmd := &cobra.Command{
		Use:   "get",
		Short: "Returns information about a scan",
		RunE:  runGetScanByIDCommand(scansWrapper),
	}

	var deleteScanID string
	deleteScanCmd := &cobra.Command{
		Use:   "delete",
		Short: "Stops a scan from running",
		RunE:  runDeleteScanCommand(deleteScanID, scansWrapper),
	}

	scanCmd.AddCommand(createScanCmd, getScanCmd, getAllScanCmd, deleteScanCmd)
	return scanCmd
}

func runCreateScanCommand(scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var input []byte
		var err error

		var incremental bool
		var verbose bool
		var scanInputFile string
		var scanInput string
		var sourcesFile string

		incremental, _ = cmd.Flags().GetBool(incrementalFlag)
		verbose, _ = cmd.Flags().GetBool(verboseFlag)
		scanInput, _ = cmd.Flags().GetString(inputFlag)
		scanInputFile, _ = cmd.Flags().GetString(inputFileFlag)
		sourcesFile, _ = cmd.Flags().GetString(sourcesFlag)

		if scanInputFile != "" {
			// Reading from input file
			PrintIfVerbose(verbose, fmt.Sprintf("Reading input from file %s", scanInputFile))
			input, err = ioutil.ReadFile(scanInputFile)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to open input file", failedCreating)
			}
		} else if scanInput != "" {
			// Reading from standard input
			PrintIfVerbose(verbose, "Reading input from console")
			input = bytes.NewBufferString(scanInput).Bytes()
		} else {
			// No input was given
			return errors.Errorf("%s: no input was given\n", failedCreating)
		}
		var scanModel = scansApi.Scan{}
		var scanResponseModel *scans.ScanResponseModel
		var errorModel *scans.ErrorModel
		// Try to parse to a scan model in order to manipulate the request payload
		err = json.Unmarshal(input, &scanModel)
		if err != nil {
			return errors.Wrapf(err, "%s: Input in bad format", failedCreating)
		}
		if sourcesFile != "" {
			// Send a request to uploads service
			var preSignedURL *string
			preSignedURL, err = uploadsWrapper.Create(sourcesFile)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to upload sources file\n", failedCreating)
			}
			// We are in upload mode - populate fields accordingly
			projectHandlerModel := scansApi.UploadProjectHandler{
				URL: *preSignedURL,
			}
			var projectHandlerModelSerialized []byte
			projectHandlerModelSerialized, err = json.Marshal(projectHandlerModel)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to upload sources file: Failed to serialize project handler",
					failedCreating)
			}
			scanModel.Project.Type = scansApi.UploadProject
			scanModel.Project.Handler = projectHandlerModelSerialized
		}
		var payload []byte
		payload, _ = json.Marshal(scanModel)
		PrintIfVerbose(verbose, fmt.Sprintf("Payload to scans service: %s\n", string(payload)))

		// Support for incremental scan
		scanModel.Config = []scansApi.Config{
			{
				Type: scansApi.SastConfig,
				Value: map[string]string{
					"incremental": strconv.FormatBool(incremental),
				},
			},
		}
		scanResponseModel, errorModel, err = scansWrapper.Create(&scanModel)
		if err != nil {
			return errors.Wrapf(err, "%s", failedCreating)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedCreating, errorModel.Code, errorModel.Message)
		} else if scanResponseModel != nil {
			fmt.Printf("Scan created successfully: Scan ID %s\n", scanResponseModel.ID)
		}
		return nil
	}
}

func runGetAllScansCommand(wrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Println("Get all scans running")
		return nil
	}
}

func runGetScanByIDCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var scanResponseModel *scans.ScanResponseModel
		var errorModel *scans.ErrorModel
		var err error
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a scan ID", failedGetting)
		}
		scanID := args[0]
		scanResponseModel, errorModel, err = scansWrapper.GetByID(scanID)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGetting)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedGetting, errorModel.Code, errorModel.Message)
		} else if scanResponseModel != nil {
			fmt.Printf("Scan ID %s:\n", scanResponseModel.ID)
			fmt.Printf("Status: %s\n", scanResponseModel.Status)
		}
		return nil
	}
}

func runDeleteScanCommand(scanID string, wrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Println("Delete scan running")
		return nil
	}
}
