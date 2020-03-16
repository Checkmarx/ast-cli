package commands

import (
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

func NewAstCLI(scansEndpoint, projectsEndpoint, resultsEndpoint string) CLI {
	rootCmd := &cobra.Command{
		Use:   "ast",
		Short: "A CLI wrapping Checkmarx AST APIs",
	}
	scanCmd := NewScanCommand(scansEndpoint)
	rootCmd.AddCommand(scanCmd)
	return &AstCLI{
		rootCmd: rootCmd,
	}
}
