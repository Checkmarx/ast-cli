package commands

import (
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
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
