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

func NewAstCLI(astApiURL, scansPath, projectsPath, uploadsPath, resultsPath string) CLI {
	rootCmd := &cobra.Command{
		Use:   "ast",
		Short: "A CLI wrapping Checkmarx AST APIs",
	}
	scansURL := fmt.Sprintf("%s/%s", astApiURL, scansPath)
	uploadsURL := fmt.Sprintf("%s/%s", astApiURL, uploadsPath)
	scanCmd := NewScanCommand(scansURL, uploadsURL)
	rootCmd.AddCommand(scanCmd)
	return &AstCLI{
		rootCmd: rootCmd,
	}
}
