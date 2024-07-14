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

func ExportSbomResults(
	exportWrapper wrappers.ExportWrapper,
	targetFile string,
	results *wrappers.ResultSummary,
	formatSbomOptions string,
) error {
	payload, err := preparePayload(results.ScanID, formatSbomOptions)
	if err != nil {
		return err
	}

	log.Println("Generating SBOM report with " + payload.FileFormat + " file format")
	sbomresp, err := exportWrapper.GenerateSbomReport(payload)
	if err != nil {
		return err
	}

	pollingResp, err := pollForCompletion(exportWrapper, sbomresp.ExportID)
	if err != nil {
		return err
	}

	if err := exportWrapper.DownloadSbomReport(pollingResp.ExportID, targetFile); err != nil {
		return errors.Wrapf(err, "Failed downloading SBOM report")
	}
	return nil
}

func preparePayload(scanID, formatSbomOptions string) (*wrappers.ExportRequestPayload, error) {
	payload := &wrappers.ExportRequestPayload{
		ScanID:     scanID,
		FileFormat: DefaultSbomOption,
	}

	if formatSbomOptions != "" && formatSbomOptions != DefaultSbomOption {
		format, err := validateSbomOptions(formatSbomOptions)
		if err != nil {
			return nil, err
		}
		payload.FileFormat = format
	}

	return payload, nil
}

func pollForCompletion(exportWrapper wrappers.ExportWrapper, exportID string) (*wrappers.ExportPollingResponse, error) {
	timeout := time.After(5 * time.Minute)
	pollingResp := &wrappers.ExportPollingResponse{ExportStatus: exportingStatus}

	for pollingResp.ExportStatus == exportingStatus || pollingResp.ExportStatus == pendingStatus {
		select {
		case <-timeout:
			return nil, errors.Errorf("SBOM generating failed - Timed out after 5 minutes")
		default:
			resp, err := exportWrapper.GetSbomReportStatus(exportID)
			if err != nil {
				return nil, errors.Wrapf(err, "failed getting SBOM report status")
			}
			pollingResp = resp
			time.Sleep(delayValueForReport * time.Second)
		}
	}

	if !strings.EqualFold(pollingResp.ExportStatus, completedStatus) {
		return nil, errors.Errorf("SBOM generating failed - Current status: %s", pollingResp.ExportStatus)
	}

	return pollingResp, nil
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
