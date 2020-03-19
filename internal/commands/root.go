package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

type CLI interface {
	Execute() error
}

type AstCLI struct {
	rootCmd *cobra.Command
}

// Execute executes the root command.
func (cli *AstCLI) Execute() error {
	return cli.rootCmd.Execute()
}

func NewAstCLI(astAPIURL, scansPath, projectsPath, uploadsPath, resultsPath string) CLI {
	var verbose bool
	rootCmd := &cobra.Command{
		Use:   "ast",
		Short: "A CLI wrapping Checkmarx AST APIs",
	}
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose mode")

	scansURL := fmt.Sprintf("%s/%s", astAPIURL, scansPath)
	uploadsURL := fmt.Sprintf("%s/%s", astAPIURL, uploadsPath)
	scanCmd := NewScanCommand(scansURL, uploadsURL, verbose)
	rootCmd.AddCommand(scanCmd)
	return &AstCLI{
		rootCmd: rootCmd,
	}
}

func PrintIfVerbose(verbose bool, msg string) {
	if verbose {
		fmt.Println(msg)
	}
}
