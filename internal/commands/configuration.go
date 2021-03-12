package commands

import (
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedSettingProp = "Failed to set property"
)

func NewConfigCommand() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "configure",
		Short: "Manage scan configurations",
	}

	storValCmd := &cobra.Command{
		Use:   "show",
		Short: "Shows current profiles configuration",
		Run: func(cmd *cobra.Command, args []string) {
			wrappers.ShowConfiguration()
		},
	}

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Sets one of the configuration properties (token, base-uri, username, password)",
		RunE:  runSetValue(),
	}
	scanCmd.AddCommand(storValCmd, setCmd)
	return scanCmd
}

func runSetValue() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		const minArgCnt = 2
		if len(args) < minArgCnt {
			return errors.Errorf("%s: Please provide a property to set and a value", failedDeleting)
		}
		if args[0] == "token" {
			wrappers.SetConfigProperty(args[0], args[1])
		} else if args[0] == "cx_base_uri" {
			wrappers.SetConfigProperty(args[0], args[1])
		} else {
			return errors.Errorf("%s: unknown property or bad value", failedSettingProp)
		}
		return nil
	}
}
