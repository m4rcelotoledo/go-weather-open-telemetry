package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Latency metrics
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)

	// Throughput metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	// Error metrics
	httpErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP errors",
		},
		[]string{"method", "endpoint", "error_type"},
	)

	// Business metrics
	cepProcessingTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cep_processing_total",
			Help: "Total number of CEP processing requests",
		},
		[]string{"cep_valid", "cep_format"},
	)

	cepValidationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cep_validation_errors_total",
			Help: "Total number of CEP validation errors",
		},
		[]string{"error_type"},
	)

	// Dependency metrics
	viacepDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "viacep_request_duration_seconds",
			Help:    "Duration of ViaCEP API requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status_code"},
	)

	weatherDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "weather_request_duration_seconds",
			Help:    "Duration of Weather API requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status_code"},
	)

	externalAPIErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "external_api_errors_total",
			Help: "Total number of external API errors",
		},
		[]string{"api_name", "error_type", "status_code"},
	)
)

// MetricsMiddleware adds Prometheus metrics to requests
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrapper to capture status code
		wrappedWriter := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Execute handler
		next.ServeHTTP(wrappedWriter, r)

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Extract request information
		method := r.Method
		endpoint := r.URL.Path
		statusCode := strconv.Itoa(wrappedWriter.statusCode)

		// Record latency metrics
		httpRequestDuration.WithLabelValues(method, endpoint, statusCode).Observe(duration)

		// Record throughput metrics
		httpRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()

		// Record error metrics
		if wrappedWriter.statusCode >= 400 {
			errorType := "client_error"
			if wrappedWriter.statusCode >= 500 {
				errorType = "server_error"
			}
			httpErrorsTotal.WithLabelValues(method, endpoint, errorType).Inc()
		}

		// Specific metrics for weather endpoint
		if endpoint == "/weather" && method == "POST" {
			// These metrics will be incremented by specific handlers
			cepProcessingTotal.WithLabelValues("unknown", "unknown").Inc()
		}
	})
}

// responseWriter wrapper to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// RecordCEPMetrics records specific CEP metrics
func RecordCEPMetrics(cep string, isValid bool, format string) {
	cepValid := "false"
	if isValid {
		cepValid = "true"
	}
	cepProcessingTotal.WithLabelValues(cepValid, format).Inc()
}

// RecordCEPValidationError records CEP validation errors
func RecordCEPValidationError(errorType string) {
	cepValidationErrors.WithLabelValues(errorType).Inc()
}

// RecordViaCEPMetrics records ViaCEP call metrics
func RecordViaCEPMetrics(duration time.Duration, statusCode int, errorType string) {
	statusCodeStr := strconv.Itoa(statusCode)

	// Duration metric
	viacepDuration.WithLabelValues(statusCodeStr).Observe(duration.Seconds())

	// Error metric
	if statusCode >= 400 {
		externalAPIErrors.WithLabelValues("viacep", errorType, statusCodeStr).Inc()
	}
}

// RecordWeatherMetrics records Weather API call metrics
func RecordWeatherMetrics(duration time.Duration, statusCode int, errorType string) {
	statusCodeStr := strconv.Itoa(statusCode)

	// Duration metric
	weatherDuration.WithLabelValues(statusCodeStr).Observe(duration.Seconds())

	// Error metric
	if statusCode >= 400 {
		externalAPIErrors.WithLabelValues("weather", errorType, statusCodeStr).Inc()
	}
}

// GetMetricsHandler returns handler to expose Prometheus metrics
func GetMetricsHandler() http.Handler {
	return promhttp.Handler()
}
