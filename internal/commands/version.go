package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	version = "2.0.0_RC1"
)

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Prints the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
}
