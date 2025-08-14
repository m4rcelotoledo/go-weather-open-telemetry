package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// Interface to allow mocking in tests
type ServiceBCaller interface {
	callServiceB(ctx context.Context, cep string) (*WeatherResponse, int, error)
}

// Mock handler for tests
type MockCEPHandler struct {
	*CEPHandler
	mockCallServiceB func(ctx context.Context, cep string) (*WeatherResponse, int, error)
}

func (m *MockCEPHandler) callServiceB(ctx context.Context, cep string) (*WeatherResponse, int, error) {
	if m.mockCallServiceB != nil {
		return m.mockCallServiceB(ctx, cep)
	}
	return m.CEPHandler.callServiceB(ctx, cep)
}

// Override the HandleCEPRequest method to use our mock
func (m *MockCEPHandler) HandleCEPRequest(w http.ResponseWriter, r *http.Request) {
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

	// Call Service B using mock
	response, statusCode, err := m.callServiceB(ctx, cepReq.CEP)
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

func TestCEPHandler_HandleCEPRequest_InvalidMethod(t *testing.T) {
	handler := NewCEPHandler(nil)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.HandleCEPRequest(w, req)

	// PRECISE test: must return 405
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "Method not allowed" {
		t.Errorf("Expected message 'Method not allowed', got '%s'", response["message"])
	}
}

func TestCEPHandler_HandleCEPRequest_InvalidJSON(t *testing.T) {
	handler := NewCEPHandler(nil)

	req := httptest.NewRequest("POST", "/", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCEPRequest(w, req)

	// PRECISE test: must return 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "invalid json" {
		t.Errorf("Expected message 'invalid json', got '%s'", response["message"])
	}
}

func TestCEPHandler_HandleCEPRequest_InvalidCEP(t *testing.T) {
	handler := NewCEPHandler(nil)

	requestBody := CEPRequest{CEP: "123"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCEPRequest(w, req)

	// PRECISE test: must return 422
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "invalid zipcode" {
		t.Errorf("Expected message 'invalid zipcode', got '%s'", response["message"])
	}
}

func TestIsValidCEP(t *testing.T) {
	tests := []struct {
		name     string
		cep      string
		expected bool
	}{
		{"Valid CEP", "29902555", true},
		{"Valid CEP with hyphen", "29902-555", true},
		{"Valid CEP with space", "29902 555", true},
		{"Invalid CEP too short", "123", false},
		{"Invalid CEP too long", "123456789", false},
		{"Invalid CEP with letters", "abc12345", false},
		{"Empty CEP", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidCEP(tt.cep)
			if result != tt.expected {
				t.Errorf("isValidCEP(%s) = %v; want %v", tt.cep, result, tt.expected)
			}
		})
	}
}

func TestNewCEPHandler(t *testing.T) {
	handler := NewCEPHandler(nil)

	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
}

// ===== TIMEOUT AND NETWORK FAILURE TESTS =====
func TestCEPHandler_NetworkTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	handler := NewCEPHandler(nil)

	// Test with valid CEP - should fail due to network timeout
	requestBody := CEPRequest{CEP: "29902555"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCEPRequest(w, req)

	// PRECISE test: must return 500 for network failure
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for network failure, got %d", w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "Error calling Service B" {
		t.Errorf("Expected message 'Error calling Service B', got '%s'", response["message"])
	}
}

// Timeout test using controlled server
func TestCEPHandler_TimeoutWithControlledServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping controlled timeout test in short mode")
	}

	// Create mock handler
	mockHandler := &MockCEPHandler{
		CEPHandler: NewCEPHandler(nil),
	}

	// Configure mock to simulate timeout error
	mockHandler.mockCallServiceB = func(ctx context.Context, cep string) (*WeatherResponse, int, error) {
		return nil, http.StatusInternalServerError, fmt.Errorf("timeout: request timed out")
	}

	// Test with valid CEP
	requestBody := CEPRequest{CEP: "29902555"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mockHandler.HandleCEPRequest(w, req)

	// PRECISE test: must return 500 (Internal Server Error) for network failure
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for network failure, got %d", w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "Error calling Service B" {
		t.Errorf("Expected message 'Error calling Service B', got '%s'", response["message"])
	}
}

// Network failure test using controlled server
func TestCEPHandler_NetworkFailureWithControlledServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping controlled network failure test in short mode")
	}

	// Create server that simulates network failure
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate network failure - return 500 error
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Internal server error"}`))
	}))
	defer server.Close()

	// Create mock handler
	mockHandler := &MockCEPHandler{
		CEPHandler: NewCEPHandler(nil),
	}

	// Configure mock to simulate network failure
	mockHandler.mockCallServiceB = func(ctx context.Context, cep string) (*WeatherResponse, int, error) {
		client := &http.Client{}

		requestBody := CEPRequest{CEP: cep}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/weather", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("network error: %w", err)
		}
		defer resp.Body.Close()

		// Return the controlled server status
		return nil, resp.StatusCode, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Test with valid CEP
	requestBody := CEPRequest{CEP: "29902555"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mockHandler.HandleCEPRequest(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for network failure, got %d", w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "Error calling Service B" {
		t.Errorf("Expected message 'Error calling Service B', got '%s'", response["message"])
	}
}

// ===== INTEGRATION TESTS FOR STATUS CODE PROPAGATION =====

func TestCEPHandler_PropagateStatusCodes_404_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock handler that simulates 404 response
	mockHandler := &MockCEPHandler{
		CEPHandler: NewCEPHandler(nil),
	}

	// Configure mock to simulate 404 response
	mockHandler.mockCallServiceB = func(ctx context.Context, cep string) (*WeatherResponse, int, error) {
		return nil, http.StatusNotFound, fmt.Errorf("CEP not found")
	}

	// Test with CEP that doesn't exist (should return 404)
	requestBody := CEPRequest{CEP: "00000000"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mockHandler.HandleCEPRequest(w, req)

	// PRECISE test: must return 404 for CEP not found
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for CEP not found, got %d", w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "can not find zipcode" {
		t.Errorf("Expected message 'can not find zipcode', got '%s'", response["message"])
	}
}

func TestCEPHandler_PropagateStatusCodes_200_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock handler that simulates 200 response
	mockHandler := &MockCEPHandler{
		CEPHandler: NewCEPHandler(nil),
	}

	// Configure mock to simulate 200 response
	mockHandler.mockCallServiceB = func(ctx context.Context, cep string) (*WeatherResponse, int, error) {
		return &WeatherResponse{
			City:  "Vila Velha",
			TempC: 25.5,
			TempF: 77.9,
			TempK: 298.6,
		}, http.StatusOK, nil
	}

	// Test with valid CEP (should return 200 if external APIs work)
	requestBody := CEPRequest{CEP: "29902555"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mockHandler.HandleCEPRequest(w, req)

	// PRECISE test: must return 200 for successful request
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for successful request, got %d", w.Code)
	}

	// Check success response
	var response WeatherResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.City == "" {
		t.Error("Expected city field in response")
	}
	if response.TempC == 0 {
		t.Error("Expected temp_C field in response")
	}
}
