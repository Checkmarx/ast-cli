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

type VorpalWrapper struct {
	port int
}

const (
	vorpalScanErrMsg = "Vorpal scan failed for file %s"
	localHostAddress = "127.0.0.1:%d"
	serviceName      = "VorpalEngine"
)

var (
	grpcClient      *ClientWithTimeout
	fullHostAddress string
)

func NewVorpalWrapper(port int) *VorpalWrapper {
	fullHostAddress = fmt.Sprintf(localHostAddress, port)
	grpcClient = NewGRPCClientWithTimeout(fullHostAddress, 1*time.Second).(*ClientWithTimeout)
	return &VorpalWrapper{
		port: port,
	}
}

// CallScan TODO: This function should move to vorpal service when it is implemented
func (s *VorpalWrapper) CallScan(filePath string) (*vorpalScan.ScanResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Printf("Error reading file %s: %v", filePath, err)
		return nil, err
	}
	_, fileName := filepath.Split(filePath)
	sourceCode := string(data)
	return s.Scan(fileName, sourceCode)
}

func (s *VorpalWrapper) Scan(fileName, sourceCode string) (*vorpalScan.ScanResult, error) {
	conn, connErr := grpcClient.CreateClientConn()
	if connErr != nil {
		logger.Printf(ConnErrMsg, s.port, connErr)
		return nil, connErr
	}

	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	scanClient := vorpalScan.NewScanServiceClient(conn)

	request := &vorpalScan.SingleScanRequest{
		ScanRequest: &vorpalScan.ScanRequest{
			Id:         uuid.New().String(),
			FileName:   fileName,
			SourceCode: sourceCode,
		},
	}

	scanResultResponse, err := scanClient.Scan(grpcClient.ctx, request)
	if err != nil {
		return nil, errors.Wrapf(err, vorpalScanErrMsg, fileName)
	}

	return scanResultResponse, nil
}

func (s *VorpalWrapper) HealthCheck() error {
	err := grpcClient.HealthCheck(serviceName)
	if err != nil {
		return err
	}
	logger.PrintIfVerbose(fmt.Sprintf("End of Health Check! Status: Serving, Host Address: %s", fullHostAddress))
	return nil
}

func (s *VorpalWrapper) ShutDown() error {
	conn, connErr := grpcClient.CreateClientConn()
	if connErr != nil {
		logger.Printf(ConnErrMsg, s.port, connErr)
		return connErr
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	managementClient := vorpalManagement.NewManagementServiceClient(conn)
	_, shutdownErr := managementClient.Shutdown(grpcClient.ctx, &vorpalManagement.ShutdownRequest{})
	if shutdownErr != nil {
		return errors.Wrap(shutdownErr, "failed to shutdown")
	}
	logger.PrintfIfVerbose("Vorpal service is shutting down")
	return nil
}
