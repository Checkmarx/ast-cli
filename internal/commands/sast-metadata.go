package commands

import (
	"encoding/json"
	"fmt"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/checkmarxDev/sast-metadata/pkg/api/v1/rest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedDownloadingEngineLog = "failed downloading engine log"
	failedGettingScanInfo      = "failed getting scan info"
	failedGettingMetrics       = "failed getting metrics"
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
		Short: "Enables the ability to manage sast scans metadata in AST",
	}
	scanInfoCmd := &cobra.Command{
		Use:   "scan-info <scan id>",
		Short: "Retrieve information about a given scan id",
		RunE:  runScanInfo(sastMetadataWrapper),
	}
	metricsCmd := &cobra.Command{
		Use:   "metrics",
		Short: "Retrieve engine metrics for a given scan id",
		RunE:  runMetrics(sastMetadataWrapper),
	}

	addFormatFlag(scanInfoCmd, FormatList, FormatJSON, FormatTable)
	addFormatFlag(metricsCmd, FormatList, FormatJSON)
	sastMetadataCmd.AddCommand(scanInfoCmd, metricsCmd)
	return sastMetadataCmd
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

func runMetrics(sastMetadataWrapper wrappers.SastMetadataWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.Errorf("%s: please provide scan id", failedGettingMetrics)
		}

		scanID := args[0]
		metrics, errorModel, err := sastMetadataWrapper.GetMetrics(scanID)
		if err != nil {
			return errors.Wrap(err, failedGettingMetrics)
		}

		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingMetrics, errorModel.Code, errorModel.Message)
		}

		f, _ := cmd.Flags().GetString(FormatFlag)
		if IsFormat(f, FormatJSON) {
			var resultsJSON []byte
			resultsJSON, err = json.Marshal(metrics)
			if err != nil {
				return errors.Wrapf(err, "%s: failed to serialize metrics response", failedGettingMetrics)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(resultsJSON))
			return nil
		}

		outputMetrics(metrics)
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

func outputMetrics(metrics *rest.Metrics) {
	fmt.Println("************ Metrics ************")
	fmt.Println("Scan id:", metrics.ScanID)
	fmt.Println("Memory peak:", metrics.MemoryPeak)
	fmt.Println("Virtual memory peak:", metrics.VirtualMemoryPeak)
	fmt.Println("Total scanned files count:", metrics.TotalScannedFilesCount)
	fmt.Println("Total scanned lines of code:", metrics.TotalScannedLOC)
	fmt.Println("Dom objects per language:")
	outputLanguagesMap(metrics.DomObjectsPerLanguage)
	fmt.Println("Successful lines of code per language:")
	outputLanguagesMap(metrics.SuccessfullLOCPerLanguage)
	fmt.Println("Failed lines of code per language:")
	outputLanguagesMap(metrics.FailedLOCPerLanguage)
	fmt.Println("File count of detected but not scanned languages")
	outputLanguagesMap(metrics.FileCountOfDetectedButNotScannedLanguages)

	fmt.Println("Scanned files per language:")
	for l, f := range metrics.ScannedFilesPerLanguage {
		fmt.Printf("%s - good files: %d, partially good files: %d, bad files: %d\n", l,
			f.GoodFiles, f.PartiallyGoodFiles, f.BadFiles)
	}
}

func outputLanguagesMap(m map[string]uint32) {
	for l, c := range m {
		fmt.Printf("%s - %d\n", l, c)
	}
}
