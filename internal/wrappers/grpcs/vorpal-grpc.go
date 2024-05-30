package grpcs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs/protos/vorpal/managements"
	scans2 "github.com/checkmarx/ast-cli/internal/wrappers/grpcs/protos/vorpal/scans"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
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
	grpcClient *ClientWithTimeout
)

func NewVorpalWrapper(port int) *VorpalWrapper {
	grpcClient = NewGRPCClientWithTimeout(fmt.Sprintf(localHostAddress, port), 1*time.Second).(*ClientWithTimeout)
	return &VorpalWrapper{
		port: port,
	}
}

// CallScan TODO: This function should move to vorpal service when it is implemented
func (s *VorpalWrapper) CallScan(filePath string) (*scans2.ScanResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Printf("Error reading file %s: %v", filePath, err)
		return nil, err
	}
	_, fileName := filepath.Split(filePath)
	sourceCode := string(data)
	return s.Scan(fileName, sourceCode)
}

func (s *VorpalWrapper) Scan(fileName, sourceCode string) (*scans2.ScanResult, error) {
	conn, connErr := grpcClient.CreateClientConn()
	if connErr != nil {
		logger.Printf(ConnErrMsg, s.port, connErr)
		return nil, connErr
	}

	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	client := scans2.NewScanServiceClient(conn)

	request := &scans2.SingleScanRequest{
		ScanRequest: &scans2.ScanRequest{
			Id:         uuid.New().String(),
			FileName:   fileName,
			SourceCode: sourceCode,
		},
	}

	scanResultResponse, err := client.Scan(grpcClient.GetContext(), request)
	if err != nil {
		return nil, errors.Wrapf(err, vorpalScanErrMsg, fileName)
	}

	return scanResultResponse, nil
}

func checkHealth(ctx context.Context, service string, conn grpc.ClientConnInterface) (*healthpb.HealthCheckResponse, error) {
	req := &healthpb.HealthCheckRequest{
		Service: service,
	}
	rsp, err := healthpb.NewHealthClient(conn).Check(ctx, req)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

func (s *VorpalWrapper) CheckHealth() error {
	conn, connErr := grpcClient.CreateClientConn()
	if connErr != nil {
		logger.Printf(ConnErrMsg, s.port, connErr)
		return connErr
	}

	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	healthRes, healthErr := checkHealth(grpcClient.GetContext(), serviceName, conn)
	if healthErr != nil {
		logger.PrintIfVerbose(fmt.Sprintf("Health Check Failed: %v", healthErr))
		return healthErr
	}

	if healthRes.Status == healthpb.HealthCheckResponse_SERVING {
		logger.PrintIfVerbose(fmt.Sprintf("End of Health Check! Status: %v, Port: %v", healthRes.Status, s.port))
		return nil
	}

	return fmt.Errorf("service not serving, status: %v", healthRes.Status)
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

	managementClient := managements.NewManagementServiceClient(conn)
	_, shutdownErr := managementClient.Shutdown(grpcClient.GetContext(), &managements.ShutdownRequest{})
	if shutdownErr != nil {
		return errors.Wrap(shutdownErr, "failed to shutdown")
	}
	logger.PrintfIfVerbose("Vorpal service is shutting down")
	return nil
}
