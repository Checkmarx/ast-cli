package scans

import (
	"context"
	"fmt"
	"os"
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

func (s *VorpalWrapper) Scan(filePath string) (*protos.ScanResult, error) {
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Print(err)
	}
	localhostAddress := fmt.Sprintf("0.0.0.0:%d", s.port)
	conn, err := grpc.NewClient(localhostAddress, options...)

	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	if err != nil {
		logger.Printf("grpc.NewClient(%q): %v", localhostAddress, err)
		return nil, errors.Wrapf(err, vorpalScanErrMsg, filePath)
	}

	client := protos.NewScanServiceClient(conn)

	request := &protos.SingleScanRequest{
		ScanRequest: &protos.ScanRequest{
			Id:         uuid.New().String(),
			FileName:   filePath,
			SourceCode: string(data),
		},
	}

	scanResultResponse, err := client.Scan(context.Background(), request)
	if err != nil {
		return nil, errors.Wrapf(err, vorpalScanErrMsg, filePath)
	}

	return scanResultResponse, nil
}

func checkHealth(service string, conn grpc.ClientConnInterface) (*healthpb.HealthCheckResponse, error) {
	req := &healthpb.HealthCheckRequest{
		Service: service,
	}
	ctx := context.Background()
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

	healthRes, healthErr := checkHealth("MicroSastEngine", conn)
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
