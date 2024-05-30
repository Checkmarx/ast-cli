package grpcs

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	ConnErrMsg = "Error occurred while creating the gRPC client at address %q: %v"
)

type Client interface {
	CreateClientConn() (*grpc.ClientConn, error)
	GetContext() context.Context
}

type BaseClient struct {
	hostAddress string
	ctx         context.Context
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

type ClientWithTimeout struct {
	BaseClient
	timeout time.Duration
}

func NewGRPCClientWithTimeout(hostAddress string, timeout time.Duration) Client {
	return &ClientWithTimeout{BaseClient: BaseClient{hostAddress: hostAddress, ctx: context.Background()}, timeout: timeout}
}

func (c *ClientWithTimeout) CreateClientConn() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()
	return c.dialContext(ctx)
}

func (c *ClientWithTimeout) GetContext() context.Context {
	return c.ctx
}
