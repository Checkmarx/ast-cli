package services

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

func GetExportPackage(exportWrapper wrappers.ExportWrapper, scanID string) (*wrappers.ScaPackageCollectionExport, error) {
	exportID, err := exportWrapper.InitiateExportRequest(scanID)
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Initiated export request for scanID %s. ExportID: %s", scanID, exportID))

	fileURL, err := exportWrapper.CheckExportStatus(exportID)
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Checked export status. File URL: %s", fileURL))

	scaPackageCollection, err := exportWrapper.GetScaPackageCollectionExport(fileURL)
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose("Retrieved SCA package collection export successfully")

	return scaPackageCollection, nil
}
