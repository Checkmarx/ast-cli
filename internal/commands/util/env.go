package util

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

const formatString = "%30v: %s\n"

func NewEnvCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Shows the current profiles configuration properties",
		Example: heredoc.Doc(
			`
			$ cx utils env
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/VJGXtw
			`,
			),
		},
		RunE: runEnvChecks(),
	}
	return cmd
}

func runEnvChecks() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("\nDetected Environment Variables:\n\n")
		for param := range properties {
			fmt.Printf(formatString, param, os.Getenv(param))
		}
		return nil
	}
}
