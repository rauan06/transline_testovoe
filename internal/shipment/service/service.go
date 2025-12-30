package service

import (
	"context"
	"fmt"

	"testovoe/internal/shipment/grpc"
	"testovoe/internal/shipment/repo"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Service struct {
	repo         *repo.Repository
	customerGrpc *grpc.Client
}

func NewService(repo *repo.Repository, customerGrpc *grpc.Client) *Service {
	return &Service{
		repo:         repo,
		customerGrpc: customerGrpc,
	}
}

type CreateShipmentRequest struct {
	Route    string
	Price    float64
	Customer struct {
		IDN string
	}
}

func (s *Service) CreateShipment(ctx context.Context, req CreateShipmentRequest) (*repo.Shipment, error) {
	ctx, span := otel.Tracer("shipment-service").Start(ctx, "CreateShipment")
	defer span.End()

	span.SetAttributes(
		attribute.String("shipment.route", req.Route),
		attribute.Float64("shipment.price", req.Price),
		attribute.String("customer.idn", req.Customer.IDN),
	)

	if len(req.Customer.IDN) != 12 {
		span.RecordError(fmt.Errorf("invalid idn length"))
		return nil, fmt.Errorf("idn must be exactly 12 digits")
	}

	customerResp, err := s.customerGrpc.UpsertCustomer(ctx, req.Customer.IDN)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to upsert customer: %w", err)
	}

	span.SetAttributes(attribute.String("customer.id", customerResp.Id))

	shipment := &repo.Shipment{
		Route:      req.Route,
		Price:      req.Price,
		Status:     "CREATED",
		CustomerID: customerResp.Id,
	}

	if err := s.repo.CreateShipment(ctx, shipment); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create shipment: %w", err)
	}

	return shipment, nil
}

func (s *Service) GetShipment(ctx context.Context, id string) (*repo.Shipment, error) {
	ctx, span := otel.Tracer("shipment-service").Start(ctx, "GetShipment")
	defer span.End()

	span.SetAttributes(attribute.String("shipment.id", id))

	shipment, err := s.repo.GetShipment(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}

	return shipment, nil
}
