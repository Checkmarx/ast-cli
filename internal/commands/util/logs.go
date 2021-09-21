package util

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const (
	ScanIDFlag   = "scan-id"
	ScanTypeFlag = "scan-type"
)

func NewLogsCommand(logsWrapper wrappers.LogsWrapper) *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Download scan log for selected engine",
		Long:  "Accepts a scan-id and engine type (sast, kics or sca) and downloads the related scan log",
		Example: heredoc.Doc(`
			$ cx utils logs --scan-id <scan Id> --scan-type <sast | sca | kics>
		`),
		RunE: runDownloadLogs(logsWrapper),
	}
	logsCmd.PersistentFlags().String(ScanIDFlag, "", "Scan ID to retrieve log for.")
	logsCmd.PersistentFlags().String(ScanTypeFlag, "", "Scan type to pull log for, ex: sast, kics or sca.")
	return logsCmd
}

func runDownloadLogs(logsWrapper wrappers.LogsWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		scanID, _ := cmd.Flags().GetString(ScanIDFlag)
		scanType, _ := cmd.Flags().GetString(ScanTypeFlag)
		log, err := logsWrapper.GetLog(scanID, scanType)
		if err != nil {
			return err
		}
		fmt.Print(log)
		return nil
	}
}
