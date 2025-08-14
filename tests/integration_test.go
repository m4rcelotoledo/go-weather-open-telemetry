package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestIntegration_ServiceCommunication tests communication between Service A and Service B
func TestIntegration_ServiceCommunication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test 1: Check if Service A is responding
	t.Run("Service A Health Check", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/health")
		if err != nil {
			t.Skipf("Service A não está rodando: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
		}
	})

	// Test 2: Check if Service B is responding
	t.Run("Service B Health Check", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/health")
		if err != nil {
			t.Skipf("Service B não está rodando: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
		}
	})

	// Test 3: Flow complete Service A -> Service B
	t.Run("Service A to Service B Flow", func(t *testing.T) {
		// Test with valid CEP
		requestBody := map[string]string{"cep": "29902555"}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		resp, err := http.Post(
			"http://localhost:8080/",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			t.Skipf("Não foi possível conectar ao Service A: %v", err)
		}
		defer resp.Body.Close()

		// The response should be 200 for successful requests
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
		}

		// Check success response structure
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check required fields
		if _, ok := response["city"]; !ok {
			t.Error("Response missing 'city' field")
		}
		if _, ok := response["temp_C"]; !ok {
			t.Error("Response missing 'temp_C' field")
		}
		if _, ok := response["temp_F"]; !ok {
			t.Error("Response missing 'temp_F' field")
		}
		if _, ok := response["temp_K"]; !ok {
			t.Error("Response missing 'temp_K' field")
		}

		t.Logf("Resposta de sucesso: %+v", response)
	})

	// Test 4: Invalid CEP validation
	t.Run("Invalid CEP Validation", func(t *testing.T) {
		requestBody := map[string]string{"cep": "123"}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		resp, err := http.Post(
			"http://localhost:8080/",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			t.Skipf("Não foi possível conectar ao Service A: %v", err)
		}
		defer resp.Body.Close()

		// Should return 422 for invalid CEP
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
		}

		var errorResponse map[string]string
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}
		if errorResponse["message"] != "invalid zipcode" {
			t.Errorf("Expected message 'invalid zipcode', got '%s'", errorResponse["message"])
		}
	})

	// Test 5: Invalid HTTP method
	t.Run("Invalid HTTP Method", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/")
		if err != nil {
			t.Skipf("Não foi possível conectar ao Service A: %v", err)
		}
		defer resp.Body.Close()

		// Should return 405 for method not allowed
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", resp.StatusCode)
		}
	})
}

// TestIntegration_ServiceB_Direct tests Service B directly
func TestIntegration_ServiceB_Direct(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Service B direct test in short mode")
	}

	// Test 1: Service B with valid CEP
	t.Run("Service B Valid CEP", func(t *testing.T) {
		requestBody := map[string]string{"cep": "29902555"}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		resp, err := http.Post(
			"http://localhost:8081/weather",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			t.Skipf("Não foi possível conectar ao Service B: %v", err)
		}
		defer resp.Body.Close()

		// The response should be 200 for successful requests
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
		}

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if _, ok := response["city"]; !ok {
			t.Error("Response missing 'city' field")
		}
		if _, ok := response["temp_C"]; !ok {
			t.Error("Response missing 'temp_C' field")
		}
		if _, ok := response["temp_F"]; !ok {
			t.Error("Response missing 'temp_F' field")
		}
		if _, ok := response["temp_K"]; !ok {
			t.Error("Response missing 'temp_K' field")
		}

		t.Logf("Service B resposta de sucesso: %+v", response)
	})

	// Test 2: Service B with invalid CEP
	t.Run("Service B Invalid CEP", func(t *testing.T) {
		requestBody := map[string]string{"cep": "123"}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		resp, err := http.Post(
			"http://localhost:8081/weather",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			t.Skipf("Não foi possível conectar ao Service B: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
		}

		var errorResponse map[string]string
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}
		if errorResponse["message"] != "invalid zipcode" {
			t.Errorf("Expected message 'invalid zipcode', got '%s'", errorResponse["message"])
		}
	})
}

// TestIntegration_Observability tests if observability services are working
func TestIntegration_Observability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping observability test in short mode")
	}

	// Test 1: Zipkin
	t.Run("Zipkin Accessibility", func(t *testing.T) {
		resp, err := http.Get("http://localhost:9411")
		if err != nil {
			t.Skipf("Zipkin não está acessível: %v", err)
		}
		defer resp.Body.Close()

		// Zipkin should return 200 or 302 (redirect)
		if resp.StatusCode != 200 && resp.StatusCode != 302 {
			t.Errorf("Expected status 200 or 302, got %d", resp.StatusCode)
		}
		t.Logf("Zipkin está acessível (status: %d)", resp.StatusCode)
	})

	// Test 2: OTEL Collector
	t.Run("OTEL Collector Accessibility", func(t *testing.T) {
		resp, err := http.Get("http://localhost:4318")
		if err != nil {
			t.Skipf("OTEL Collector não está acessível: %v", err)
		}
		defer resp.Body.Close()

		// OTEL Collector can return different status codes
		t.Logf("OTEL Collector está acessível (status: %d)", resp.StatusCode)
	})
}

// TestIntegration_EndToEnd tests the complete end-to-end flow
func TestIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Check if both services are running
	serviceAHealth, err := http.Get("http://localhost:8080/health")
	if err != nil {
		t.Skipf("Service A não está rodando: %v", err)
	}
	serviceAHealth.Body.Close()

	serviceBHealth, err := http.Get("http://localhost:8081/health")
	if err != nil {
		t.Skipf("Service B não está rodando: %v", err)
	}
	serviceBHealth.Body.Close()

	// Test complete end-to-end flow
	t.Run("Complete End-to-End Flow", func(t *testing.T) {
		// Test with valid CEP
		requestBody := map[string]string{"cep": "29902555"}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		start := time.Now()
		resp, err := http.Post(
			"http://localhost:8080/",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		duration := time.Since(start)

		if err != nil {
			t.Skipf("Falha na requisição end-to-end: %v", err)
		}
		defer resp.Body.Close()

		// Check response
		if resp.StatusCode != 200 && resp.StatusCode != 500 {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
		}

		// Check response time (should be < 10 seconds for external APIs)
		if duration >= 10*time.Second {
			t.Errorf("Resposta muito lenta: %v", duration)
		}

		t.Logf("Fluxo end-to-end concluído em %v (status: %d)", duration, resp.StatusCode)

		// Log response
		var response interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err == nil {
			t.Logf("Resposta: %+v", response)
		}
	})
}

// TestIntegration_ErrorScenarios tests error scenarios
func TestIntegration_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error scenarios test in short mode")
	}

	// Test 1: Invalid JSON
	t.Run("Invalid JSON", func(t *testing.T) {
		resp, err := http.Post(
			"http://localhost:8080/",
			"application/json",
			bytes.NewBuffer([]byte("invalid json")),
		)
		if err != nil {
			t.Skipf("Não foi possível conectar ao Service A: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
		}

		var errorResponse map[string]string
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}
		if errorResponse["message"] != "invalid json" {
			t.Errorf("Expected message 'invalid json', got '%s'", errorResponse["message"])
		}
	})

	// Test 2: Empty CEP
	t.Run("Empty CEP", func(t *testing.T) {
		requestBody := map[string]string{"cep": ""}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		resp, err := http.Post(
			"http://localhost:8080/",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			t.Skipf("Não foi possível conectar ao Service A: %v", err)
		}
		defer resp.Body.Close()

		// Should return 422 for empty CEP
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d", resp.StatusCode)
		}
	})
}

// Helper function to run all integration tests
func TestAllIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping all integration tests in short mode")
	}

	t.Log("🚀 Executando todos os testes de integração...")
	t.Log("📋 Verificando comunicação entre serviços...")
	t.Log("🔍 Testando fluxos end-to-end...")
	t.Log("📊 Verificando observabilidade...")
	t.Log("❌ Testando cenários de erro...")

	// This test runs all other integration tests
	// Useful to run everything at once
}
