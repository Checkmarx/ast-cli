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
	EngineLogDestFile          = "engine-log.txt"
	failedDownloadingEngineLog = "failed downloading engine log"
)

func NewSSICommand(ssiWrapper wrappers.SSIWrapper) *cobra.Command {
	ssiCmd := &cobra.Command{
		Use:   "ssi",
		Short: "Manage sast scan incremental data",
	}
	downloadEngineLogCmd := &cobra.Command{
		Use:   "download-engine-log <scan id>",
		Short: "Download the engine log for given scan id",
		RunE:  runDownloadEngineLog(ssiWrapper),
	}
	ssiCmd.AddCommand(downloadEngineLogCmd)
	return ssiCmd
}

func runDownloadEngineLog(ssiWrapper wrappers.SSIWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide scan id", failedDownloadingEngineLog)
		}

		scanID := args[0]
		logReader, errorModel, err := ssiWrapper.DownloadEngineLog(scanID)
		if err != nil {
			return errors.Wrap(err, failedDownloadingEngineLog)
		}

		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedDownloadingEngineLog, errorModel.Code, errorModel.Message)
		}

		defer logReader.Close()
		pwdDir, err := os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "%s: failed get current directory path", failedDownloadingEngineLog)
		}

		destFile := filepath.Join(pwdDir, EngineLogDestFile)
		destWriter, err := os.Create(destFile)
		if err != nil {
			return errors.Wrapf(err, "%s failed creating file to download into it", failedDownloadingEngineLog)
		}

		defer destWriter.Close()
		_, _ = cmd.OutOrStdout().Write([]byte("Downloading into " + destFile + "\n"))
		_, err = io.Copy(destWriter, logReader)
		if err != nil {
			return errors.Wrap(err, failedDownloadingEngineLog)
		}

		return nil
	}
}
