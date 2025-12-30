package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "testovoe/api/proto"
	"testovoe/internal/customer/service"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Server struct {
	pb.UnimplementedCustomerServiceServer
	service *service.Service
}

func NewServer(svc *service.Service) *Server {
	return &Server{service: svc}
}

func (s *Server) UpsertCustomer(ctx context.Context, req *pb.UpsertCustomerRequest) (*pb.CustomerResponse, error) {
	ctx, span := otel.Tracer("customer-grpc").Start(ctx, "UpsertCustomer")
	defer span.End()

	span.SetAttributes(
		attribute.String("grpc.method", "UpsertCustomer"),
		attribute.String("customer.idn", req.Idn),
	)

	c, err := s.service.UpsertCustomer(ctx, req.Idn)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to upsert customer: %v", err)
	}

	if spanCtx := span.SpanContext(); spanCtx.IsValid() {
		traceID := spanCtx.TraceID().String()
		log.Printf("Upserted customer %s (idn: %s), trace_id: %s", c.ID, c.IDN, traceID)
	}

	return &pb.CustomerResponse{
		Id:        c.ID,
		Idn:       c.IDN,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) GetCustomer(ctx context.Context, req *pb.GetCustomerRequest) (*pb.CustomerResponse, error) {
	ctx, span := otel.Tracer("customer-grpc").Start(ctx, "GetCustomer")
	defer span.End()

	span.SetAttributes(
		attribute.String("grpc.method", "GetCustomer"),
		attribute.String("customer.idn", req.Idn),
	)

	c, err := s.service.GetCustomer(ctx, req.Idn)
	if err != nil {
		span.RecordError(err)
		if err.Error() == "customer not found" {
			return nil, status.Errorf(codes.NotFound, "customer not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get customer: %v", err)
	}

	if spanCtx := span.SpanContext(); spanCtx.IsValid() {
		traceID := spanCtx.TraceID().String()
		log.Printf("Retrieved customer %s (idn: %s), trace_id: %s", c.ID, c.IDN, traceID)
	}

	return &pb.CustomerResponse{
		Id:        c.ID,
		Idn:       c.IDN,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}, nil
}

func StartGRPCServer(port string, svc *service.Service) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
	)

	pb.RegisterCustomerServiceServer(s, NewServer(svc))

	log.Printf("Customer gRPC server listening on :%s", port)
	return s.Serve(lis)
}
