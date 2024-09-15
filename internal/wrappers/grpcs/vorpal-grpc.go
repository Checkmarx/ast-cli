package grpcs

import (
	"fmt"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type ASCAGrpcWrapper struct {
	grpcClient  *ClientWithTimeout
	hostAddress string
	port        int
	serving     bool
}

const (
	ASCAScanErrMsg   = "ASCA scan failed for file %s. ScanId: %s"
	localHostAddress = "127.0.0.1:%d"
	serviceName      = "ASCAEngine"
)

func NewASCAGrpcWrapper(port int) ASCAWrapper {
	serverHostAddress := fmt.Sprintf(localHostAddress, port)
	return &ASCAGrpcWrapper{
		grpcClient:  NewGRPCClientWithTimeout(serverHostAddress, 1*time.Second).(*ClientWithTimeout),
		hostAddress: serverHostAddress,
		port:        port,
	}
}

func (v *ASCAGrpcWrapper) Scan(fileName, sourceCode string) (*ScanResult, error) {
	conn, connErr := v.grpcClient.CreateClientConn()
	if connErr != nil {
		logger.Printf(ConnErrMsg, v.hostAddress, connErr)
		return nil, connErr
	}

	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	scanClient := ASCAScan.NewScanServiceClient(conn)
	scanID := uuid.New().String()

	request := &ASCAScan.SingleScanRequest{
		ScanRequest: &ASCAScan.ScanRequest{
			Id:         scanID,
			FileName:   fileName,
			SourceCode: sourceCode,
		},
	}

	resp, err := scanClient.Scan(v.grpcClient.ctx, request)
	if err != nil {
		return nil, errors.Wrapf(err, ASCAScanErrMsg, fileName, scanID)
	}

	var scanError *Error
	if resp.Error != nil {
		scanError = &Error{
			Code:        ErrorCode(resp.Error.Code),
			Description: resp.Error.Description,
		}
	}
	return &ScanResult{
		RequestID:   resp.RequestId,
		Status:      resp.Status,
		Message:     resp.Message,
		ScanDetails: convertScanDetails(resp.ScanDetails),
		Error:       scanError,
	}, nil
}

func convertScanDetails(details []*ASCAScan.ScanResult_ScanDetail) []ScanDetail {
	var scanDetails []ScanDetail
	for _, detail := range details {
		scanDetails = append(scanDetails, ScanDetail{
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

func (v *ASCAGrpcWrapper) HealthCheck() error {
	if !v.serving {
		err := v.grpcClient.HealthCheck(v.grpcClient, serviceName)
		if err != nil {
			return err
		}
		logger.PrintIfVerbose(fmt.Sprintf("End of Health Check. Status: Serving, Host Address: %v", v.hostAddress))
		v.serving = true
	}
	return nil
}

func (v *ASCAGrpcWrapper) ShutDown() error {
	conn, connErr := v.grpcClient.CreateClientConn()
	if connErr != nil {
		logger.Printf(ConnErrMsg, v.hostAddress, connErr)
		return connErr
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	managementClient := ASCAManagement.NewManagementServiceClient(conn)
	_, shutdownErr := managementClient.Shutdown(v.grpcClient.ctx, &ASCAManagement.ShutdownRequest{})
	if shutdownErr != nil {
		return errors.Wrap(shutdownErr, "failed to shutdown")
	}
	logger.PrintfIfVerbose("ASCA service is shutting down")
	v.serving = false
	return nil
}

func (v *ASCAGrpcWrapper) GetPort() int {
	return v.port
}

func (v *ASCAGrpcWrapper) ConfigurePort(port int) {
	v.port = port
	v.hostAddress = fmt.Sprintf(localHostAddress, port)
	v.grpcClient = NewGRPCClientWithTimeout(v.hostAddress, 1*time.Second).(*ClientWithTimeout)
	v.serving = false
}
