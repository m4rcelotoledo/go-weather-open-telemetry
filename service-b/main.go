package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"service-b/internal/clients/viacep"
	"service-b/internal/clients/weather"
	"service-b/internal/config"
	"service-b/internal/handlers"
	"service-b/internal/middleware"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","time":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
}

func initTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// Create OTLP exporter
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("otel-collector:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("service-b"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global TracerProvider and TextMapPropagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Check if WEATHER_API_KEY is set
	if cfg.WeatherAPIKey == "" {
		log.Fatal("WEATHER_API_KEY environment variable is required")
	}

	// Initialize OpenTelemetry
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Create clients
	viaCEPClient := viacep.NewClient()
	weatherClient := weather.NewClient()

	// Create handler
	orchestratorHandler := handlers.NewOrchestratorHandler(viaCEPClient, weatherClient, cfg)

	// Create mux
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/weather", orchestratorHandler.HandleWeatherRequest)
	mux.HandleFunc("/metrics", middleware.GetMetricsHandler().ServeHTTP)

	// Apply middleware
	handler := middleware.MetricsMiddleware(mux)

	// Create server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Service B starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Service B...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Service B exited")
}
