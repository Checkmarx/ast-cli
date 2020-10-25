package commands

import (
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
)

const (
	failedDownloadingEngineLog = "failed downloading engine log"
)

func NewSastMetadataCommand(sastMetadataWrapper wrappers.SastMetadataWrapper) *cobra.Command {
	sastMetadataCmd := &cobra.Command{
		Use:   "sast-metadata",
		Short: "Manage sast metadata on scans",
	}
	downloadEngineLogCmd := &cobra.Command{
		Use:   "engine-log <scan id>",
		Short: "Downloads the engine log for given scan id",
		RunE:  runDownloadEngineLog(sastMetadataWrapper),
	}
	sastMetadataCmd.AddCommand(downloadEngineLogCmd)
	return sastMetadataCmd
}

func runDownloadEngineLog(sastMetadataWrapper wrappers.SastMetadataWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide scan id", failedDownloadingEngineLog)
		}

		scanID := args[0]
		logReader, errorModel, err := sastMetadataWrapper.DownloadEngineLog(scanID)
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
