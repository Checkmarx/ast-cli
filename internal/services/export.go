package services

import (
	"log"
	"strings"
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

const (
	DefaultSbomOption   = "CycloneDxJson"
	exportingStatus     = "Exporting"
	delayValueForReport = 10
	pendingStatus       = "Pending"
	completedStatus     = "completed"
)

func ExportSbomResults(exportWrapper wrappers.ExportWrapper,
	targetFile string,
	results *wrappers.ResultSummary,
	formatSbomOptions string) error {
	payload := &wrappers.ExportRequestPayload{
		ScanID:     results.ScanID,
		FileFormat: DefaultSbomOption,
	}
	if formatSbomOptions != "" && formatSbomOptions != DefaultSbomOption {
		format, err := validateSbomOptions(formatSbomOptions)
		if err != nil {
			return err
		}
		payload.FileFormat = format
	}
	pollingResp := &wrappers.ExportPollingResponse{}

	sbomresp, err := exportWrapper.GenerateSbomReport(payload)
	if err != nil {
		return err
	}

	log.Println("Generating SBOM report with " + payload.FileFormat + " file format")
	pollingResp.ExportStatus = exportingStatus
	for pollingResp.ExportStatus == exportingStatus || pollingResp.ExportStatus == pendingStatus {
		pollingResp, err = exportWrapper.GetSbomReportStatus(sbomresp.ExportID)
		if err != nil {
			return errors.Wrapf(err, "%s", "failed getting SBOM report status")
		}
		time.Sleep(delayValueForReport * time.Second)
	}
	if !strings.EqualFold(pollingResp.ExportStatus, completedStatus) {
		return errors.Errorf("SBOM generating failed - Current status: %s", pollingResp.ExportStatus)
	}
	err = exportWrapper.DownloadSbomReport(pollingResp.ExportID, targetFile)
	if err != nil {
		return errors.Wrapf(err, "%s", "Failed downloading SBOM report")
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
