package services

import (
	"strings"
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

const (
	DefaultSbomOption          = "CycloneDxJson"
	exportingStatus            = "Exporting"
	delayValueForReport        = 10
	pendingStatus              = "Pending"
	completedStatus            = "completed"
	fiveMinutes                = 5 * time.Minute
	sbomGenerationTimeoutError = "SBOM generation timed out"
)

func ExportSbomResults(exportWrapper wrappers.ExportWrapper, targetFile string, results *wrappers.ResultSummary, formatSbomOptions string) error {
	format, err := getFormat(formatSbomOptions)
	if err != nil {
		return err
	}

	sbomResp, err := generateSbomReport(exportWrapper, results.ScanID, format)
	if err != nil {
		return err
	}

	if err := waitForSbomReport(exportWrapper, sbomResp.ExportID); err != nil {
		return err
	}

	if err := exportWrapper.DownloadSbomReport(sbomResp.ExportID, targetFile); err != nil {
		return errors.Wrapf(err, "%s", "Failed downloading SBOM report")
	}

	return nil
}

func getFormat(formatSbomOptions string) (string, error) {
	if formatSbomOptions == "" || formatSbomOptions == DefaultSbomOption {
		return DefaultSbomOption, nil
	}
	return validateSbomOptions(formatSbomOptions)
}

func generateSbomReport(exportWrapper wrappers.ExportWrapper, scanID, format string) (*wrappers.ExportResponse, error) {
	payload := &wrappers.ExportRequestPayload{
		ScanID:     scanID,
		FileFormat: format,
	}
	return exportWrapper.GenerateSbomReport(payload)
}

func waitForSbomReport(exportWrapper wrappers.ExportWrapper, exportID string) error {
	pollingResp := &wrappers.ExportPollingResponse{ExportStatus: exportingStatus}
	timeoutTicker := time.NewTicker(fiveMinutes)
	defer timeoutTicker.Stop()

	pollingTicker := time.NewTicker(delayValueForReport * time.Second)
	defer pollingTicker.Stop()

	for pollingResp.ExportStatus == exportingStatus || pollingResp.ExportStatus == pendingStatus {
		select {
		case <-pollingTicker.C:
			var err error
			pollingResp, err = exportWrapper.GetSbomReportStatus(exportID)
			if err != nil {
				return errors.Wrap(err, "failed getting SBOM report status")
			}
		case <-timeoutTicker.C:
			return errors.New(sbomGenerationTimeoutError)
		}

		if strings.EqualFold(pollingResp.ExportStatus, completedStatus) {
			return errors.Errorf("SBOM generating failed - Current status: %s", pollingResp.ExportStatus)
		}
	}
	return nil
}

func validateSbomOptions(sbomOption string) (string, error) {
	var sbomOptionsStringMap = map[string]string{
		"cyclonedxjson": "CycloneDxJson",
		"cyclonedxxml":  "CycloneDxXml",
		"spdxjson":      "SpdxJson",
	}
	sbomOption = strings.ToLower(strings.ReplaceAll(sbomOption, " ", ""))
	if sbomOptionsStringMap[sbomOption] != "" {
		return sbomOptionsStringMap[sbomOption], nil
	}
	return "", errors.Errorf("invalid SBOM option: %s", sbomOption)
}
