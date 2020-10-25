package commands

import (
	"io"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/checkmarxDev/sast-scan-inc/pkg/api/v1/rest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedDownloadingEngineLog = "failed downloading engine log"
	failedGettingScanInfo      = "failed getting scan info"
)

type ScanInfoView struct {
	ScanID    string `format:"name:Scan id"`
	ProjectID string `format:"name:Project id"`
	FileCount int32  `format:"name:File count"`
	Loc       int32
	Type      string
}

type IncScanInfoView struct {
	*ScanInfoView
	BaseID            string  `format:"name:Base id"`
	CanceledReason    string  `format:"name:Canceled reason;omitempty"`
	AddedFilesCount   int32   `format:"name:Added files count"`
	ChangedFilesCount int32   `format:"name:Changed files count"`
	DeletedFilesCount int32   `format:"name:Deleted files count"`
	ChangePercentage  float32 `format:"name:Change percentage"`
}

func NewSastMetadataCommand(sastMetadataWrapper wrappers.SastMetadataWrapper) *cobra.Command {
	sastMetadataCmd := &cobra.Command{
		Use:   "sast-metadata",
		Short: "Manage sast metadata on scans",
	}
	engineLogCmd := &cobra.Command{
		Use:   "engine-log <scan id>",
		Short: "Downloads the engine log for given scan id",
		RunE:  runEngineLog(sastMetadataWrapper),
	}
	scanInfoCmd := &cobra.Command{
		Use:   "scan-info",
		Short: "Gets information for given scan id",
		RunE:  runScanInfo(sastMetadataWrapper),
	}
	addFormatFlag(scanInfoCmd, formatTable, formatJSON, formatList)
	sastMetadataCmd.AddCommand(engineLogCmd, scanInfoCmd)
	return sastMetadataCmd
}

func runEngineLog(sastMetadataWrapper wrappers.SastMetadataWrapper) func(*cobra.Command, []string) error {
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

func runScanInfo(sastMetadataWrapper wrappers.SastMetadataWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide scan id", failedGettingScanInfo)
		}

		scanID := args[0]
		scanInfo, errorModel, err := sastMetadataWrapper.GetScanInfo(scanID)
		if err != nil {
			return errors.Wrap(err, failedGettingScanInfo)
		}

		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingScanInfo, errorModel.Code, errorModel.Message)
		}

		err = printByFormat(cmd, toScanInfoView(scanInfo))
		if err != nil {
			return errors.Wrap(err, failedGettingScanInfo)
		}

		return nil
	}
}

func toScanInfoView(info *rest.ScanInfo) interface{} {
	s := &ScanInfoView{
		ScanID:    info.ScanID,
		ProjectID: info.ProjectID,
		Loc:       info.Loc,
		FileCount: info.FileCount,
	}

	if info.IsIncremental {
		s.Type = "incremental scan"
		return &IncScanInfoView{
			ScanInfoView: s,
			BaseID:       info.BaseID,
			CanceledReason: func() string {
				if info.IsIncrementalCanceled {
					return info.IncrementalCancelReason
				}

				return ""
			}(),
			AddedFilesCount:   info.AddedFilesCount,
			ChangedFilesCount: info.ChangedFilesCount,
			DeletedFilesCount: info.DeletedFilesCount,
			ChangePercentage:  info.ChangePercentage,
		}
	}

	s.Type = "full scan"
	return s
}
