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

type Customer struct {
	ID        string
	IDN       string
	CreatedAt time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertCustomer(ctx context.Context, idn string) (*Customer, error) {
	ctx, span := otel.Tracer("customer-repo").Start(ctx, "UpsertCustomer")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "upsert"),
		attribute.String("customer.idn", idn),
	)

	// Сначала пытаемся найти существующего клиента
	var customer Customer
	query := `SELECT id, idn, created_at FROM customers WHERE idn = $1`
	err := r.db.QueryRowContext(ctx, query, idn).Scan(
		&customer.ID,
		&customer.IDN,
		&customer.CreatedAt,
	)

	if err == nil {
		span.SetAttributes(attribute.String("db.result", "found"))
		return &customer, nil
	}

	if err != sql.ErrNoRows {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to query customer: %w", err)
	}

	// Если не найден, создаём нового
	span.SetAttributes(attribute.String("db.result", "created"))
	customer.ID = uuid.New().String()
	customer.IDN = idn
	customer.CreatedAt = time.Now()

	insertQuery := `INSERT INTO customers (id, idn, created_at) VALUES ($1, $2, $3) 
		ON CONFLICT (idn) DO UPDATE SET idn = EXCLUDED.idn 
		RETURNING id, idn, created_at`
	
	err = r.db.QueryRowContext(ctx, insertQuery, customer.ID, customer.IDN, customer.CreatedAt).Scan(
		&customer.ID,
		&customer.IDN,
		&customer.CreatedAt,
	)

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to insert customer: %w", err)
	}

	return &customer, nil
}

func (r *Repository) GetCustomer(ctx context.Context, idn string) (*Customer, error) {
	ctx, span := otel.Tracer("customer-repo").Start(ctx, "GetCustomer")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "select"),
		attribute.String("customer.idn", idn),
	)

	var customer Customer
	query := `SELECT id, idn, created_at FROM customers WHERE idn = $1`
	err := r.db.QueryRowContext(ctx, query, idn).Scan(
		&customer.ID,
		&customer.IDN,
		&customer.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			span.SetAttributes(attribute.String("db.result", "not_found"))
			return nil, fmt.Errorf("customer not found")
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to query customer: %w", err)
	}

	span.SetAttributes(attribute.String("db.result", "found"))
	return &customer, nil
}

func (r *Repository) GetCustomerByID(ctx context.Context, id string) (*Customer, error) {
	ctx, span := otel.Tracer("customer-repo").Start(ctx, "GetCustomerByID")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "select"),
		attribute.String("customer.id", id),
	)

	var customer Customer
	query := `SELECT id, idn, created_at FROM customers WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&customer.ID,
		&customer.IDN,
		&customer.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			span.SetAttributes(attribute.String("db.result", "not_found"))
			return nil, fmt.Errorf("customer not found")
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to query customer: %w", err)
	}

	span.SetAttributes(attribute.String("db.result", "found"))
	return &customer, nil
}

