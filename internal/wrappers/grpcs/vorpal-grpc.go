package grpcs

import (
	"fmt"
	"os"
	"path/filepath"
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
	}
}

// CreateVorpalScanRequest TODO: This function should move to vorpal service when it is implemented
func (v *VorpalGrpcWrapper) CreateVorpalScanRequest(filePath string) (*vorpalScan.ScanResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Printf("Error reading file %v: %v", filePath, err)
		return nil, err
	}
	_, fileName := filepath.Split(filePath)
	sourceCode := string(data)
	return v.Scan(fileName, sourceCode)
}

func (v *VorpalGrpcWrapper) Scan(fileName, sourceCode string) (*vorpalScan.ScanResult, error) {
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

	scanResultResponse, err := scanClient.Scan(v.grpcClient.ctx, request)
	if err != nil {
		return nil, errors.Wrapf(err, vorpalScanErrMsg, fileName, scanID)
	}

	return scanResultResponse, nil
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
