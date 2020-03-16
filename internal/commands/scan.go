package commands

import (
	"fmt"

	http "github.com/checkmarxDev/ast-cli/internal/http"
	"github.com/spf13/cobra"
)

func NewScanCommand(endpoint string) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage AST scans",
	}
	scansWrapper := http.NewHTTPScansWrapper(endpoint)
	createScanCmd := &cobra.Command{
		Use:   "create",
		Short: "Creates and runs a new scan",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Create scan running")
			fmt.Println(scansWrapper)
		},
	}
	scanCmd.AddCommand(createScanCmd)
	return scanCmd
}

func init() {

}
