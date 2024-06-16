package grpcs

import (
	"fmt"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	vorpalManagement "github.com/checkmarx/ast-cli/internal/wrappers/grpcs/protos/vorpal/managements"
	vorpalScan "github.com/checkmarx/ast-cli/internal/wrappers/grpcs/protos/vorpal/scans"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type VorpalGrpcWrapper struct {
	grpcClient  *ClientWithTimeout
	hostAddress string
	port        int
}

const (
	vorpalScanErrMsg = "Vorpal scan failed for file %s. ScanId: %s"
	localHostAddress = "127.0.0.1:%d"
	serviceName      = "VorpalEngine"
)

func NewVorpalGrpcWrapper(port int) VorpalWrapper {
	serverHostAddress := fmt.Sprintf(localHostAddress, port)
	return &VorpalGrpcWrapper{
		grpcClient:  NewGRPCClientWithTimeout(serverHostAddress, 1*time.Second).(*ClientWithTimeout),
		hostAddress: serverHostAddress,
		port:        port,
	}
}

func (v *VorpalGrpcWrapper) Scan(fileName, sourceCode string) (*ScanResult, error) {
	conn, connErr := v.grpcClient.CreateClientConn()
	if connErr != nil {
		logger.Printf(ConnErrMsg, v.hostAddress, connErr)
		return nil, connErr
	}

	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	scanClient := vorpalScan.NewScanServiceClient(conn)
	scanID := uuid.New().String()

	request := &vorpalScan.SingleScanRequest{
		ScanRequest: &vorpalScan.ScanRequest{
			Id:         scanID,
			FileName:   fileName,
			SourceCode: sourceCode,
		},
	}

	resp, err := scanClient.Scan(v.grpcClient.ctx, request)
	if err != nil {
		return nil, errors.Wrapf(err, vorpalScanErrMsg, fileName, scanID)
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

func convertScanDetails(details []*vorpalScan.ScanResult_ScanDetail) []ScanDetail {
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

func (v *VorpalGrpcWrapper) HealthCheck() error {
	err := v.grpcClient.HealthCheck(v.grpcClient, serviceName)
	if err != nil {
		return err
	}
	logger.PrintIfVerbose(fmt.Sprintf("End of Health Check. Status: Serving, Host Address: %v", v.hostAddress))
	return nil
}

func (v *VorpalGrpcWrapper) ShutDown() error {
	conn, connErr := v.grpcClient.CreateClientConn()
	if connErr != nil {
		logger.Printf(ConnErrMsg, v.hostAddress, connErr)
		return connErr
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	managementClient := vorpalManagement.NewManagementServiceClient(conn)
	_, shutdownErr := managementClient.Shutdown(v.grpcClient.ctx, &vorpalManagement.ShutdownRequest{})
	if shutdownErr != nil {
		return errors.Wrap(shutdownErr, "failed to shutdown")
	}
	logger.PrintfIfVerbose("Vorpal service is shutting down")
	return nil
}

func (v *VorpalGrpcWrapper) GetPort() int {
	return v.port
}
