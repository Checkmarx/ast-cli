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
	limitFlag       = "limit"
	limitFlagSh     = "l"
	limitUsage      = "The number of items to return"
	offsetFlag      = "offset"
	offsetFlagSh    = "o"
	offsetUsage     = "The number of items to skip before collecting the results"
)

// Return an AST CLI root command to execute
func NewAstCLI(scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	resultsWrapper wrappers.ResultsWrapper) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ast",
		Short: "A CLI wrapping Checkmarx AST APIs",
	}
	rootCmd.PersistentFlags().BoolP(verboseFlag, verboseFlagSh, false, "Verbose mode")

	scanCmd := NewScanCommand(scansWrapper, uploadsWrapper)
	projectCmd := NewProjectCommand(projectsWrapper)
	resultCmd := NewResultCommand(resultsWrapper)
	versionCmd := NewVersionCommand()
	rootCmd.AddCommand(scanCmd, projectCmd, resultCmd, versionCmd)
	rootCmd.SilenceUsage = true
	return rootCmd
}

func PrintIfVerbose(verbose bool, msg string) {
	if verbose {
		fmt.Println(msg)
	}
}

func getLimitAndOffset(cmd *cobra.Command) (limit, offset uint64) {
	verbose, _ := cmd.Flags().GetBool(verboseFlag)
	limit, _ = cmd.Flags().GetUint64(limitFlag)
	offset, _ = cmd.Flags().GetUint64(offsetFlag)
	PrintIfVerbose(verbose, fmt.Sprintf("%s: %d", limitFlag, limit))
	PrintIfVerbose(verbose, fmt.Sprintf("%s: %d", offsetFlag, offset))
	return
}
