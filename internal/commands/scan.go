package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	clihttp "github.com/checkmarxDev/ast-cli/internal/http"
	scansApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
	"github.com/checkmarxDev/scans/pkg/scans"
	"github.com/spf13/cobra"
)

func NewScanCommand(scansURL, uploadsURL string) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage AST scans",
	}

	scansWrapper := clihttp.NewHTTPScansWrapper(scansURL)
	uploadsWrapper := clihttp.NewUploadsHttpWrapper(uploadsURL)
	var scanInput string
	var scanInputFile string
	var sourcesFile string
	createScanCmd := &cobra.Command{
		Use:   "create",
		Short: "Creates and runs a new scan",
		Run: func(cmd *cobra.Command, args []string) {
			var input []byte
			var err error
			if scanInputFile != "" {
				// Reading from input file
				fmt.Printf("Reading input from file %s\n", scanInputFile)
				input, err = ioutil.ReadFile(scanInputFile)
				if err != nil {
					fmt.Printf("Failed to open input file: %s\n", err.Error())
					return
				}
			} else if scanInput != "" {
				// Reading from standard input
				input = bytes.NewBufferString(scanInput).Bytes()
			} else {
				// No input was given
				fmt.Println("Failed creating a scan: no input was given")
				return
			}
			var scanModel = scansApi.Scan{}
			var scanReponseModel *scans.ScanResponseModel
			var errorModel *scans.ErrorModel
			//Try to parse to a scan model in order to manipulate the request payload
			err = json.Unmarshal(input, &scanModel)
			if err != nil {
				fmt.Printf("Failed creating a scan: Input in bad format - %s", err.Error())
				return
			}
			if sourcesFile != "" {
				//Send a request to uploads service
				var preSignedURL *string
				preSignedURL, err = uploadsWrapper.Create(sourcesFile)
				if err != nil {
					fmt.Printf("Failed to upload sources file: %s\n", err.Error())
					return
				}
				// We are in upload mode - populate fields accordingly
				projectHandlerModel := scansApi.UploadProjectHandler{
					URL: *preSignedURL,
				}
				var projectHandlerModelSerialized []byte
				projectHandlerModelSerialized, err = json.Marshal(projectHandlerModel)
				if err != nil {
					fmt.Printf("Failed to upload sources file: Failed to serialize project handler - %s\n", err.Error())
					return
				}
				scanModel.Project.Type = scansApi.UploadProject
				scanModel.Project.Handler = projectHandlerModelSerialized
				// TODO remove
				var payload []byte
				payload, _ = json.Marshal(scanModel)
				fmt.Printf("Payload to scans service: %s\n", string(payload))
			}

			scanReponseModel, errorModel, err = scansWrapper.Create(&scanModel)
			if err != nil {
				fmt.Printf("Failed creating a scan: %s", err.Error())
				return
			}
			// TODO remove
			fmt.Println(scanReponseModel)
			fmt.Println(errorModel)
		},
	}
	createScanCmd.Flags().StringVarP(&sourcesFile, "sources", "s", "", "A path to the sources file to scan")
	createScanCmd.Flags().StringVarP(&scanInput, "input", "i", "", "The object representing the requested scan, in JSON format")
	createScanCmd.Flags().StringVarP(&scanInputFile, "inputFile", "f", "", "A file holding the requested scan object in JSON format. Takes precedence over --input")

	getScanCmd := &cobra.Command{
		Use:   "get",
		Short: "Returns information about a scan",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Get scan running")
		},
	}
	deleteScanCmd := &cobra.Command{
		Use:   "delete",
		Short: "Stops a scan from running",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Delete scan running")
		},
	}
	scanCmd.AddCommand(createScanCmd, getScanCmd, deleteScanCmd)
	return scanCmd
}
