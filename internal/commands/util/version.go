package util

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Prints the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(params.Version)
		},
	}
}
