package scans

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers/vorpal/scans/protos"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type VorpalWrapper struct {
	port int
}

var (
	vorpalScanErrMsg = "Vorpal scan failed for file %s"
	grpcConnErrMsg   = "Error occurred while creating the grpc client in address %q. error: %v"
)

func NewVorpalWrapper(port int) *VorpalWrapper {
	return &VorpalWrapper{
		port: port,
	}
}

// TODO: This function should move to vorpal service when it is implemented
func (s *VorpalWrapper) callScan(filePath string) (*protos.ScanResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Printf("Error reading file %s: %v", filePath, err)
		return nil, err
	}
	_, fileName := filepath.Split(filePath)
	sourceCode := string(data)
	return s.Scan(fileName, sourceCode)
}

func (s *VorpalWrapper) Scan(fileName, sourceCode string) (*protos.ScanResult, error) {
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	timeout := 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	localhostAddress := fmt.Sprintf("0.0.0.0:%d", s.port)
	conn, err := grpc.DialContext(ctx, localhostAddress, options...)
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	if err != nil {
		logger.Printf("grpc.DialContext(%q): %v", localhostAddress, err)
		return nil, errors.Wrapf(err, vorpalScanErrMsg, fileName)
	}

	client := protos.NewScanServiceClient(conn)

	request := &protos.SingleScanRequest{
		ScanRequest: &protos.ScanRequest{
			Id:         uuid.New().String(),
			FileName:   fileName,
			SourceCode: sourceCode,
		},
	}

	scanResultResponse, err := client.Scan(ctx, request)
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
	timeout := 1 * time.Second
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	localhostAddress := fmt.Sprintf("0.0.0.0:%d", s.port)
	conn, err := grpc.DialContext(ctx, localhostAddress, options...)
	if err != nil {
		logger.Printf(grpcConnErrMsg, localhostAddress, err)
		return err
	}

	defer func(conn *grpc.ClientConn) {
		if conn != nil {
			_ = conn.Close()
		}
	}(conn)

	healthRes, healthErr := checkHealth(ctx, "MicroSastEngine", conn)
	if healthErr != nil {
		logger.Printf("Health Check Failed: %v", healthErr)
		return healthErr
	}

	if healthRes.Status == healthpb.HealthCheckResponse_SERVING {
		logger.Printf("End of Health Check! Status: %v", healthRes.Status)
		return nil
	}

	return fmt.Errorf("service not serving, status: %v", healthRes.Status)
}
