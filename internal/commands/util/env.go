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
				https://checkmarx.com/resource/documents/en/34965-68653-utils.html#UUID-f7245425-72b9-9854-a60a-a9f37e0173d9
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
		for param := range Properties {
			fmt.Printf(formatString, param, os.Getenv(param))
		}
		return nil
	}
}
