package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	pb "testovoe/api/proto"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.CustomerServiceClient
}

func NewClient(endpoint string) (*Client, error) {
	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to customer service: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewCustomerServiceClient(conn),
	}, nil
}

func (c *Client) UpsertCustomer(ctx context.Context, idn string) (*pb.CustomerResponse, error) {
	req := &pb.UpsertCustomerRequest{Idn: idn}
	return c.client.UpsertCustomer(ctx, req)
}

func (c *Client) GetCustomer(ctx context.Context, idn string) (*pb.CustomerResponse, error) {
	req := &pb.GetCustomerRequest{Idn: idn}
	return c.client.GetCustomer(ctx, req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}

