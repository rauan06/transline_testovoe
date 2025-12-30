package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"testovoe/internal/shipment/grpc"
	httphandler "testovoe/internal/shipment/http"
	"testovoe/internal/shipment/repo"
	"testovoe/internal/shipment/service"
)

func initTracer() func() {
	ctx := context.Background()

	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "localhost:4317"
	} else {
		// Убираем http:// префикс если есть
		otelEndpoint = strings.TrimPrefix(otelEndpoint, "http://")
		otelEndpoint = strings.TrimPrefix(otelEndpoint, "https://")
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("failed to create OTLP exporter: %v", err)
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "shipment-service"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}
}

func main() {
	// Инициализация OpenTelemetry
	shutdown := initTracer()
	defer shutdown()

	// Подключение к БД
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "testovoe"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	log.Println("Connected to database")

	// Подключение к customer-service через Envoy
	grpcEndpoint := os.Getenv("GRPC_ENVOY_ENDPOINT")
	if grpcEndpoint == "" {
		grpcEndpoint = "localhost:9090"
	}

	customerGrpc, err := grpc.NewClient(grpcEndpoint)
	if err != nil {
		log.Fatalf("failed to create customer gRPC client: %v", err)
	}
	defer customerGrpc.Close()

	log.Printf("Connected to customer service via Envoy at %s", grpcEndpoint)

	// Инициализация слоёв
	repo := repo.NewRepository(db)
	svc := service.NewService(repo, customerGrpc)
	handler := httphandler.NewHandler(svc)

	// Настройка HTTP роутера
	router := mux.NewRouter()
	router.Use(otelmux.Middleware("shipment-service"))

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/shipments", handler.CreateShipment).Methods("POST")
	api.HandleFunc("/shipments/{id}", handler.GetShipment).Methods("GET")

	// Запуск HTTP сервера
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: router,
	}

	go func() {
		log.Printf("Shipment HTTP server listening on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start HTTP server: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}
}
