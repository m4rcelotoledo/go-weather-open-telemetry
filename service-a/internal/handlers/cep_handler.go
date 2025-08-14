package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type CEPHandler struct {
}

type CEPRequest struct {
	CEP string `json:"cep"`
}

type WeatherResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func NewCEPHandler(cfg interface{}) *CEPHandler {
	return &CEPHandler{}
}

func (h *CEPHandler) HandleCEPRequest(w http.ResponseWriter, r *http.Request) {
	// Get tracer
	tracer := otel.Tracer("service-a")
	ctx, span := tracer.Start(r.Context(), "handle-weather-request")
	defer span.End()

	// Add request method to span
	span.SetAttributes(attribute.String("http.method", r.Method))
	span.SetAttributes(attribute.String("http.url", r.URL.String()))

	// Validate HTTP method
	if r.Method != "POST" {
		span.SetAttributes(attribute.String("error", "method_not_allowed"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method not allowed"})
		return
	}

	// Parse request body
	var cepReq CEPRequest
	if err := json.NewDecoder(r.Body).Decode(&cepReq); err != nil {
		log.Printf("Error decoding request: %v", err)
		span.SetAttributes(attribute.String("error", "invalid_json"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid json"})
		return
	}

	// Add CEP to span
	span.SetAttributes(attribute.String("cep", cepReq.CEP))

	// Validate CEP (8 numeric digits)
	if !isValidCEP(cepReq.CEP) {
		log.Printf("Invalid CEP format: %s", cepReq.CEP)
		span.SetAttributes(attribute.String("error", "invalid_cep"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid zipcode"})
		return
	}

	// Call Service B
	response, statusCode, err := h.callServiceB(ctx, cepReq.CEP)
	if err != nil {
		log.Printf("Error calling Service B: %v", err)
		span.SetAttributes(attribute.String("error", "service_b_error"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		// Try to extract the original error message from Service B
		if statusCode == http.StatusNotFound {
			json.NewEncoder(w).Encode(map[string]string{"message": "can not find zipcode"})
		} else {
			json.NewEncoder(w).Encode(map[string]string{"message": "Error calling Service B"})
		}
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *CEPHandler) callServiceB(ctx context.Context, cep string) (*WeatherResponse, int, error) {
	tracer := otel.Tracer("service-a")
	ctx, span := tracer.Start(ctx, "call-service-b")
	defer span.End()

	span.SetAttributes(attribute.String("cep", cep))
	span.SetAttributes(attribute.String("service_b_url", "http://service-b:8081/weather"))

	// Prepare request to Service B
	requestBody := CEPRequest{CEP: cep}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		span.SetAttributes(attribute.String("error", "json_marshal_error"))
		return nil, http.StatusInternalServerError, fmt.Errorf("error marshaling request: %w", err)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "POST", "http://service-b:8081/weather", bytes.NewBuffer(jsonBody))
	if err != nil {
		span.SetAttributes(attribute.String("error", "request_creation_error"))
		return nil, http.StatusInternalServerError, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		span.SetAttributes(attribute.String("error", "http_request_error"))
		return nil, http.StatusInternalServerError, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.SetAttributes(attribute.String("error", "response_read_error"))
		return nil, http.StatusInternalServerError, fmt.Errorf("error reading response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		span.SetAttributes(attribute.String("error", "service_b_error"))
		return nil, resp.StatusCode, fmt.Errorf("service B returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var weatherResp WeatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		span.SetAttributes(attribute.String("error", "response_unmarshal_error"))
		return nil, http.StatusInternalServerError, fmt.Errorf("error unmarshaling response: %w", err)
	}

	span.SetAttributes(attribute.String("city", weatherResp.City))
	span.SetAttributes(attribute.Float64("temp_c", weatherResp.TempC))

	return &weatherResp, http.StatusOK, nil
}

func isValidCEP(cep string) bool {
	// Remove spaces and hyphens
	cleaned := strings.ReplaceAll(strings.ReplaceAll(cep, "-", ""), " ", "")
	matched, _ := regexp.MatchString(`^\d{8}$`, cleaned)
	return matched
}
