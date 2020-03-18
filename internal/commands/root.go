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
	rootCmd := &cobra.Command{
		Use:   "ast",
		Short: "A CLI wrapping Checkmarx AST APIs",
	}
	scansURL := fmt.Sprintf("%s/%s", astAPIURL, scansPath)
	uploadsURL := fmt.Sprintf("%s/%s", astAPIURL, uploadsPath)
	scanCmd := NewScanCommand(scansURL, uploadsURL)
	rootCmd.AddCommand(scanCmd)
	return &AstCLI{
		rootCmd: rootCmd,
	}
}
