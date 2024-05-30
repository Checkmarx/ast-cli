package grpcs

import (
	"context"
	"fmt"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	ConnErrMsg = "Error occurred while creating the gRPC client at address %q: %v"
)

type Client interface {
	CreateClientConn() (*grpc.ClientConn, error)
	HealthCheck(serviceName string) error
}

type BaseClient struct {
	hostAddress string
	ctx         context.Context
}

type ClientWithTimeout struct {
	BaseClient
	timeout time.Duration
}

func NewGRPCClientWithTimeout(hostAddress string, timeout time.Duration) Client {
	return &ClientWithTimeout{BaseClient: BaseClient{hostAddress: hostAddress, ctx: context.Background()}, timeout: timeout}
}

func dialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}
}

func (c *BaseClient) dialContext(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, c.hostAddress, dialOptions()...)
}

func (c *BaseClient) HealthCheck(serviceName string) error {
	conn, connErr := c.CreateClientConn()
	if connErr != nil {
		return connErr
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	req := &healthpb.HealthCheckRequest{
		Service: serviceName,
	}

	healthRes, healthErr := healthpb.NewHealthClient(conn).Check(c.ctx, req)
	if healthErr != nil {
		logger.PrintIfVerbose(fmt.Sprintf("Health Check Failed: %v", healthErr))
		return healthErr
	}

	if healthRes.Status == healthpb.HealthCheckResponse_SERVING {
		return nil
	}

	return fmt.Errorf("service not serving, status: %v", healthRes.Status)
}

func (c *ClientWithTimeout) CreateClientConn() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()
	return c.dialContext(ctx)
}

func (c *BaseClient) CreateClientConn() (*grpc.ClientConn, error) {
	return c.dialContext(c.ctx)
}
