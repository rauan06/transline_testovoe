package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Shipment struct {
	ID         string
	Route      string
	Price      float64
	Status     string
	CustomerID string
	CreatedAt  time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateShipment(ctx context.Context, shipment *Shipment) error {
	ctx, span := otel.Tracer("shipment-repo").Start(ctx, "CreateShipment")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "insert"),
		attribute.String("shipment.route", shipment.Route),
		attribute.String("shipment.customer_id", shipment.CustomerID),
	)

	if shipment.ID == "" {
		shipment.ID = uuid.New().String()
	}
	if shipment.Status == "" {
		shipment.Status = "CREATED"
	}
	if shipment.CreatedAt.IsZero() {
		shipment.CreatedAt = time.Now()
	}

	query := `INSERT INTO shipments (id, route, price, status, customer_id, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6)`
	
	_, err := r.db.ExecContext(ctx, query,
		shipment.ID,
		shipment.Route,
		shipment.Price,
		shipment.Status,
		shipment.CustomerID,
		shipment.CreatedAt,
	)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to insert shipment: %w", err)
	}

	span.SetAttributes(attribute.String("db.result", "created"))
	return nil
}

func (r *Repository) GetShipment(ctx context.Context, id string) (*Shipment, error) {
	ctx, span := otel.Tracer("shipment-repo").Start(ctx, "GetShipment")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "select"),
		attribute.String("shipment.id", id),
	)

	var shipment Shipment
	query := `SELECT id, route, price, status, customer_id, created_at 
		FROM shipments WHERE id = $1`
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&shipment.ID,
		&shipment.Route,
		&shipment.Price,
		&shipment.Status,
		&shipment.CustomerID,
		&shipment.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			span.SetAttributes(attribute.String("db.result", "not_found"))
			return nil, fmt.Errorf("shipment not found")
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to query shipment: %w", err)
	}

	span.SetAttributes(attribute.String("db.result", "found"))
	return &shipment, nil
}

