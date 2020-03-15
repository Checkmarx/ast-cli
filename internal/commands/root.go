package commands

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "ast",
		Short: "A CLI wrapping Checkmarx AST APIs",
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

// docker build -t ast-cli .
// docker run --env-file ./config.env ast-cli
