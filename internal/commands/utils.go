package commands

import (
	"fmt"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewUtilsCommand(healthCheckWrapper wrappers.HealthCheckWrapper,
	ssiWrapper wrappers.SastMetadataWrapper,
	rmWrapper wrappers.SastRmWrapper,
	logsWrapper wrappers.LogsWrapper,
	queriesWrapper wrappers.QueriesWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "utils",
		Short: "AST Utility functions",
	}
	healthCheckCmd := NewHealthCheckCommand(healthCheckWrapper)
	ssiCmd := NewSastMetadataCommand(ssiWrapper)
	rmCmd := NewSastResourcesCommand(rmWrapper)
	queriesCmd := NewQueryCommand(queriesWrapper, uploadsWrapper)
	logsCmd := NewLogsCommand(logsWrapper)
	configCmd := NewConfigCommand()
	scanCmd.AddCommand(healthCheckCmd, ssiCmd, rmCmd, queriesCmd, logsCmd, configCmd)
	return scanCmd
}

func runTestCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if 1 == 2 {
			return errors.Errorf("%s: Please provide a scan ID", "message")
		}
		fmt.Println("Print a message")
		return nil
	}
}
