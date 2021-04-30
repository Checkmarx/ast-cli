package commands

import (
	"io"
	"os"
	"path/filepath"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedDownloadingLogs = "failed downloading"
	LogsDestFileName      = "logs.zip"
)

func NewLogsCommand(logsWrapper wrappers.LogsWrapper) *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Manage logs",
	}
	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "download zip file containing system logs",
		RunE:  runDownloadLogs(logsWrapper),
	}
	sastEngineLogCmd := &cobra.Command{
		Use:   "sast <scan id>",
		Short: "Print the sast engine log for a given scan id",
		RunE:  runEngineLog(logsWrapper, "sast"),
	}

	logsCmd.AddCommand(downloadCmd, sastEngineLogCmd)
	return logsCmd
}

func runDownloadLogs(logsWrapper wrappers.LogsWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		reader, errModel, err := logsWrapper.GetURL()
		if err != nil {
			return errors.Wrap(err, failedDownloadingLogs)
		}

		if errModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedDownloadingLogs, errModel.Code, errModel.Message)
		}

		defer reader.Close()
		pwdDir, err := os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "%s: failed get current directory path", failedDownloadingLogs)
		}

		destFile := filepath.Join(pwdDir, LogsDestFileName)
		destWriter, err := os.Create(destFile)
		if err != nil {
			return errors.Wrapf(err, "%s failed creating file to download into it", failedDownloadingLogs)
		}

		defer destWriter.Close()
		_, _ = cmd.OutOrStdout().Write([]byte("Downloading into " + destFile + "\n"))
		_, err = io.Copy(destWriter, reader)
		if err != nil {
			return errors.Wrap(err, failedDownloadingLogs)
		}

		return nil
	}
}

func runEngineLog(sastMetadataWrapper wrappers.LogsWrapper, engine string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide scan id", failedDownloadingEngineLog)
		}

		scanID := args[0]
		logReader, errorModel, err := sastMetadataWrapper.DownloadEngineLog(scanID, engine)
		if err != nil {
			return errors.Wrap(err, failedDownloadingEngineLog)
		}

		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedDownloadingEngineLog, errorModel.Code, errorModel.Message)
		}

		defer logReader.Close()
		_, err = io.Copy(cmd.OutOrStdout(), logReader)
		if err != nil {
			return errors.Wrap(err, failedDownloadingEngineLog)
		}

		return nil
	}
}
