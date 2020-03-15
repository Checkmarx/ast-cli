package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of AST",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("AST CLI Version 1")
	},
}
