package http

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"testovoe/internal/shipment/service"
)

type Handler struct {
	service *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{service: svc}
}

type CreateShipmentRequest struct {
	Route    string  `json:"route"`
	Price    float64 `json:"price"`
	Customer struct {
		IDN string `json:"idn"`
	} `json:"customer"`
}

type ShipmentResponse struct {
	ID        string    `json:"id"`
	Route     string    `json:"route"`
	Price     float64   `json:"price"`
	Status    string    `json:"status"`
	CustomerID string   `json:"customerId"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) CreateShipment(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("shipment-http").Start(r.Context(), "CreateShipment")
	defer span.End()

	span.SetAttributes(attribute.String("http.method", r.Method))
	span.SetAttributes(attribute.String("http.path", r.URL.Path))

	var req CreateShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	shipment, err := h.service.CreateShipment(ctx, service.CreateShipmentRequest{
		Route:    req.Route,
		Price:    req.Price,
		Customer: struct{ IDN string }{IDN: req.Customer.IDN},
	})

	if err != nil {
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := ShipmentResponse{
		ID:         shipment.ID,
		Route:      shipment.Route,
		Price:      shipment.Price,
		Status:     shipment.Status,
		CustomerID: shipment.CustomerID,
		CreatedAt:  shipment.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	// Логирование с trace_id
	if spanCtx := span.SpanContext(); spanCtx.IsValid() {
		traceID := spanCtx.TraceID().String()
		log.Printf("Created shipment %s, trace_id: %s", shipment.ID, traceID)
	}
}

func (h *Handler) GetShipment(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("shipment-http").Start(r.Context(), "GetShipment")
	defer span.End()

	vars := mux.Vars(r)
	id := vars["id"]

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.path", r.URL.Path),
		attribute.String("shipment.id", id),
	)

	shipment, err := h.service.GetShipment(ctx, id)
	if err != nil {
		span.RecordError(err)
		if err.Error() == "shipment not found" {
			http.Error(w, "shipment not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := ShipmentResponse{
		ID:         shipment.ID,
		Route:      shipment.Route,
		Price:      shipment.Price,
		Status:     shipment.Status,
		CustomerID: shipment.CustomerID,
		CreatedAt:  shipment.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Логирование с trace_id
	if spanCtx := span.SpanContext(); spanCtx.IsValid() {
		traceID := spanCtx.TraceID().String()
		log.Printf("Retrieved shipment %s, trace_id: %s", shipment.ID, traceID)
	}
}

