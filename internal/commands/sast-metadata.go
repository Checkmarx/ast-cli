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
	ScanID            string `format:"name:Scan id"`
	ProjectID         string `format:"name:Project id"`
	FileCount         int32  `format:"name:File count"`
	Loc               int32  `format:"name:Lines of code"`
	QueryPreset       string `format:"name:Query Preset"`
	Type              string
	BaseID            string      `format:"name:Base id;omitempty"`
	CanceledReason    string      `format:"name:Canceled reason;omitempty"`
	AddedFilesCount   interface{} `format:"name:Added files count;omitempty"`
	ChangedFilesCount interface{} `format:"name:Changed files count;omitempty"`
	DeletedFilesCount interface{} `format:"name:Deleted files count;omitempty"`
	ChangePercentage  interface{} `format:"name:Change percentage;omitempty"`
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
	addFormatFlag(scanInfoCmd, formatList, formatJSON, formatTable)
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

func valueOrNil(val interface{}, condition bool) interface{} {
	if condition {
		return val
	}

	return nil
}

func toScanInfoView(info *rest.ScanInfo) *ScanInfoView {
	hasIncrementalFields := info.IsIncremental || info.IsIncrementalCanceled
	return &ScanInfoView{
		ScanID:      info.ScanID,
		ProjectID:   info.ProjectID,
		Loc:         info.Loc,
		FileCount:   info.FileCount,
		QueryPreset: info.QueryPreset,
		Type: func() string {
			if info.IsIncremental {
				return "incremental scan"
			}

			if info.IsIncrementalCanceled {
				return "full scan - incremental canceled"
			}

			return "full scan"
		}(),
		BaseID:            info.BaseID,
		CanceledReason:    info.IncrementalCancelReason,
		AddedFilesCount:   valueOrNil(info.AddedFilesCount, hasIncrementalFields),
		ChangedFilesCount: valueOrNil(info.ChangedFilesCount, hasIncrementalFields),
		DeletedFilesCount: valueOrNil(info.DeletedFilesCount, hasIncrementalFields),
		ChangePercentage:  valueOrNil(info.ChangePercentage, hasIncrementalFields),
	}
}
