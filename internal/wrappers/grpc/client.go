package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	LocalhostAddress = "0.0.0.0:%d"
	ConnErrMsg       = "Error occurred while creating the gRPC client at address %q: %v"
)

type Client interface {
	CreateClientConn() (*grpc.ClientConn, error)
	GetContext() context.Context
}

type BaseClient struct {
	port int
	ctx  context.Context
}

func (c *BaseClient) dialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}
}

func (c *BaseClient) dialContext(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, fmt.Sprintf(LocalhostAddress, c.port), c.dialOptions()...)
}

type ClientWithTimeout struct {
	BaseClient
	timeout time.Duration
}

func NewGRPCClientWithTimeout(port int, timeout time.Duration) Client {
	return &ClientWithTimeout{BaseClient: BaseClient{port: port, ctx: context.Background()}, timeout: timeout}
}

func (c *ClientWithTimeout) CreateClientConn() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()
	return c.dialContext(ctx)
}

func (c *ClientWithTimeout) GetContext() context.Context {
	return c.ctx
}
