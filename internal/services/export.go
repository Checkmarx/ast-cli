package services

import (
	"fmt"
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

const defaultSbomOption = "CycloneDxJson"

func GetExportPackage(exportWrapper wrappers.ExportWrapper, scanID string) (*wrappers.ScaPackageCollectionExport, error) {
	var scaPackageCollection = &wrappers.ScaPackageCollectionExport{
		Packages: []wrappers.ScaPackage{},
		ScaTypes: []wrappers.ScaType{},
	}
	exportID, err := exportWrapper.InitiateExportRequest(scanID, "ScanReportJson")
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Initiated export request for scanID %s. ExportID: %s", scanID, exportID))

	fileURL, err := exportWrapper.CheckExportStatus(exportID)
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Checked export status. File URL: %s", fileURL))
	if fileURL != "" {
		scaPackageCollection, err = exportWrapper.GetScaPackageCollectionExport(fileURL)
	}
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose("Retrieved SCA package collection export successfully")

	return scaPackageCollection, nil
}

func ExportSbomResults(exportWrapper wrappers.ExportWrapper,
	targetFile string,
	results *wrappers.ResultSummary,
	formatSbomOptions string) error {
	fileFormat := defaultSbomOption
	if formatSbomOptions != "" && formatSbomOptions != defaultSbomOption {
		format, err := validateSbomOptions(formatSbomOptions)
		if err != nil {
			return err
		}
		fileFormat = format
	}

	exportID, err := exportWrapper.InitiateExportRequest(results.ScanID, fileFormat)
	if err != nil {
		return err
	}

	reportURL, err := exportWrapper.CheckExportStatus(exportID)
	if err != nil {
		return err
	}

	err = exportWrapper.DownloadExportReport(reportURL, targetFile)
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
