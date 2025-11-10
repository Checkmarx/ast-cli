package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

const (
	DefaultSbomOption   = "CycloneDxJson"
	exportingStatus     = "Exporting"
	delayValueForReport = 3
	pendingStatus       = "Pending"
	completedStatus     = "Completed"
	pollingTimeout      = 15 // minutes
)

func GetExportPackage(exportWrapper wrappers.ExportWrapper, scanID string, scaHideDevAndTestDep bool, featureflagWrappers wrappers.FeatureFlagsWrapper) (*wrappers.ScaPackageCollectionExport, error) {
	var scaPackageCollection = &wrappers.ScaPackageCollectionExport{
		Packages: []wrappers.ScaPackage{},
		ScaTypes: []wrappers.ScaType{},
	}
	payload := &wrappers.ExportRequestPayload{
		ScanID:     scanID,
		FileFormat: "ScanReportJson",
		ExportParameters: wrappers.ExportParameters{
			HideDevAndTestDependencies: scaHideDevAndTestDep,
		},
	}

	exportID, err := exportWrapper.InitiateExportRequest(payload)
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Initiated export request for scanID %s. ExportID: %s", scanID, exportID))

	exportResponse, err := pollForCompletion(exportWrapper, exportID.ExportID)

	if err != nil {
		return nil, err
	}
	minioEnabled, _ := wrappers.GetSpecificFeatureFlag(featureflagWrappers, wrappers.MinioEnabled)

	if exportResponse != nil && strings.EqualFold(exportResponse.ExportStatus, completedStatus) && exportResponse.FileURL != "" {
		filePath := ""
		if minioEnabled.Status {
			filePath = exportResponse.FileURL
		} else {
			filePath = exportID.ExportID
		}
		scaPackageCollection, err = exportWrapper.GetScaPackageCollectionExport(filePath, minioEnabled.Status)
		if err != nil {
			return nil, err
		}
	}

	return scaPackageCollection, nil
}

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
	sbomresp, err := exportWrapper.InitiateExportRequest(payload)
	if err != nil {
		return errors.Wrapf(err, "Failed downloading SBOM report")
	}

	pollingResp, err := pollForCompletion(exportWrapper, sbomresp.ExportID)
	if err != nil {
		return errors.Wrapf(err, "Failed downloading SBOM report")
	}

	if pollingResp == nil {
		log.Println("No results were found, skipping SBOM report generation")
		return nil
	}

	if err := exportWrapper.DownloadExportReport(pollingResp.ExportID, targetFile); err != nil {
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
	timeout := time.After(pollingTimeout * time.Minute)
	pollingResp := &wrappers.ExportPollingResponse{ExportStatus: exportingStatus}

	logger.PrintIfVerbose("Polling for export report generation completion")

	for pollingResp.ExportStatus == exportingStatus || pollingResp.ExportStatus == pendingStatus {

		logger.Printf("SCA Export Status is: %s", pollingResp.ExportStatus)

		select {
		case <-timeout:
			return nil, errors.Errorf("export generating failed - Timed out after 5 minutes")
		default:
			resp, err := exportWrapper.GetExportReportStatus(exportID)
			if err != nil {
				return nil, errors.Wrapf(err, "failed getting export report status")
			}
			pollingResp = resp
			logger.PrintIfVerbose("Export status: " + pollingResp.ExportStatus)
			time.Sleep(delayValueForReport * time.Second)
		}
	}

	if pollingResp.ErrorMessage != "" {
		if strings.Contains(pollingResp.ErrorMessage, "No results were found") {
			return nil, nil
		}
		return nil, fmt.Errorf("export error: %s", pollingResp.ErrorMessage)
	}

	if !strings.EqualFold(pollingResp.ExportStatus, completedStatus) {
		return nil, errors.Errorf("export generating failed - Current status: %s", pollingResp.ExportStatus)
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
