package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/checkmarxDev/scans/pkg/scans"

	http "github.com/checkmarxDev/ast-cli/internal/http"
	"github.com/spf13/cobra"
)

func NewScanCommand(endpoint string) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage AST scans",
	}
	scansWrapper := http.NewHTTPScansWrapper(endpoint)

	var scanInput string
	var scanInputFile string
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
					fmt.Printf("Failed opening input file: %s\n", err.Error())
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
			var scanModel *scans.ScanResponseModel
			var errorModel *scans.ErrorModel
			scanModel, errorModel, err = scansWrapper.Create(input)
			if err != nil {
				fmt.Printf("Failed ceating a scan: %s", err.Error())
				return
			}
			fmt.Println(scanModel)
			fmt.Println(errorModel)
		},
	}
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
	fmt.Println(scansWrapper)
	return scanCmd
}

func init() {

}
