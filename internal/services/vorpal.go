package services

import (
	"os"
	"path/filepath"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	vorpalScan "github.com/checkmarx/ast-cli/internal/wrappers/grpcs/protos/vorpal/scans"
)

func CreateVorpalScanRequest(vorpalWrapper grpcs.VorpalWrapper, filePath string) (*grpcs.ScanResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Printf("Error reading file %v: %v", filePath, err)
		return nil, err
	}
	_, fileName := filepath.Split(filePath)
	sourceCode := string(data)
	resp, err := vorpalWrapper.Scan(fileName, sourceCode)
	if err != nil {
		return nil, err
	}
	return &grpcs.ScanResult{
		RequestID:   resp.RequestId,
		Status:      resp.Status,
		Message:     resp.Message,
		ScanDetails: convertScanDetails(resp.ScanDetails),
		Error: &grpcs.Error{
			Code:        grpcs.ErrorCode(resp.Error.Code),
			Description: resp.Error.Description,
		},
	}, nil
}

func convertScanDetails(details []*vorpalScan.ScanResult_ScanDetail) []grpcs.ScanDetail {
	var scanDetails []grpcs.ScanDetail
	for _, detail := range details {
		scanDetails = append(scanDetails, grpcs.ScanDetail{
			RuleID:          detail.RuleId,
			RuleName:        detail.RuleName,
			Language:        detail.Language,
			Severity:        detail.Severity,
			FileName:        detail.FileName,
			Line:            detail.Line,
			ProblematicLine: detail.ProblematicLine,
			Length:          detail.Length,
			Remediation:     detail.RemediationAdvise,
			Description:     detail.Description,
		})
	}
	return scanDetails
}
