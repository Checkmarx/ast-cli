package commands

import (
	"fmt"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const (
	verboseFlag     = "verbose"
	verboseFlagSh   = "v"
	sourcesFlag     = "sources"
	sourcesFlagSh   = "s"
	inputFlag       = "input"
	inputFlagSh     = "i"
	inputFileFlag   = "inputFile"
	inputFileFlagSh = "f"
	incrementalFlag = "incremental"
)

// Return an AST CLI root command to execute
func NewAstCLI(scansWrapper wrappers.ScansWrapper, uploadsWrapper wrappers.UploadsWrapper) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ast",
		Short: "A CLI wrapping Checkmarx AST APIs",
	}
	rootCmd.PersistentFlags().BoolP(verboseFlag, verboseFlagSh, false, "Verbose mode")

	scanCmd := NewScanCommand(scansWrapper, uploadsWrapper)
	rootCmd.AddCommand(scanCmd)
	return rootCmd
}

func PrintIfVerbose(verbose bool, msg string) {
	if verbose {
		fmt.Println(msg)
	}
}

func createASTCommand() *cobra.Command {
	scansMockWrapper := &wrappers.ScansMockWrapper{}
	uploadsMockWrapper := &wrappers.UploadsMockWrapper{}
	return NewAstCLI(scansMockWrapper, uploadsMockWrapper)
}
