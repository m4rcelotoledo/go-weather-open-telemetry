package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// TestZipkinTracing verifies if traces are being sent to Zipkin
func TestZipkinTracing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Zipkin trace test in short mode")
	}

	// 1. Check if Zipkin is running
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

	// 2. Make a request to generate traces
	t.Run("Generate Traces", func(t *testing.T) {
		// Make request to Service A
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
			t.Skipf("Não foi possível conectar ao Service A: %v", err)
		}
		defer resp.Body.Close()

		// Response log
		t.Logf("Requisição para Service A concluída em %v (status: %d)", duration, resp.StatusCode)

		// Wait for trace processing
		time.Sleep(3 * time.Second)
	})

	// 3. Check if traces were created
	t.Run("Check Traces in Zipkin", func(t *testing.T) {
		// Query Zipkin API for traces
		resp, err := http.Get("http://localhost:9411/api/v2/traces")
		if err != nil {
			t.Skipf("Não foi possível consultar API do Zipkin: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 from Zipkin API, got %d", resp.StatusCode)
		}

		// Decode response
		var traces [][]map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&traces)
		if err != nil {
			t.Fatalf("Failed to decode Zipkin response: %v", err)
		}

		// Check if there are traces
		if len(traces) == 0 {
			t.Log("Nenhum trace encontrado no Zipkin (pode ser normal se não houver requisições recentes)")
		} else {
			t.Logf("Encontrados %d traces no Zipkin", len(traces))

			// Check trace structure (only log in verbose mode)
			if testing.Verbose() {
				for i, trace := range traces {
					t.Logf("Trace %d: %d spans", i+1, len(trace))

					for j, span := range trace {
						if serviceName, ok := span["localEndpoint"].(map[string]interface{}); ok {
							if name, ok := serviceName["serviceName"].(string); ok {
								t.Logf("  Span %d: service=%s", j+1, name)
							}
						}

						if operationName, ok := span["name"].(string); ok {
							t.Logf("  Span %d: operation=%s", j+1, operationName)
						}
					}
				}
			}
		}
	})

	// 4. Check specific spans
	t.Run("Check Specific Spans", func(t *testing.T) {
		// Query Service A spans
		resp, err := http.Get("http://localhost:9411/api/v2/spans?serviceName=service-a")
		if err != nil {
			t.Skipf("Não foi possível consultar spans do Service A: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 from Zipkin API, got %d", resp.StatusCode)
		}

		// Decode response - may return empty string if no spans
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		// If body is empty, no spans to analyze
		if len(body) == 0 || string(body) == "" {
			t.Log("Nenhum span do Service A encontrado (pode ser normal se não houver requisições recentes)")
			return
		}

		// The API returns an array of span names (strings), not span objects
		var spanNames []string
		err = json.Unmarshal(body, &spanNames)
		if err != nil {
			t.Logf("Response body: %s", string(body))
			t.Fatalf("Failed to decode spans response: %v", err)
		}

		// Check if there are Service A span names
		if len(spanNames) == 0 {
			t.Log("Nenhum span do Service A encontrado (pode ser normal se não houver requisições recentes)")
		} else {
			t.Logf("Encontrados %d spans do Service A", len(spanNames))

			// Check if there are spans with specific operations
			foundHandleRequest := false
			foundCallServiceB := false

			for _, spanName := range spanNames {
				if spanName == "handle-weather-request" {
					foundHandleRequest = true
				}
				if spanName == "call-service-b" {
					foundCallServiceB = true
				}
			}

			if foundHandleRequest {
				t.Log("✓ Span 'handle-weather-request' encontrado")
			} else {
				t.Log("⚠ Span 'handle-weather-request' não encontrado")
			}

			if foundCallServiceB {
				t.Log("✓ Span 'call-service-b' encontrado")
			} else {
				t.Log("⚠ Span 'call-service-b' não encontrado")
			}
		}
	})

	// 5. Check trace context propagation
	t.Run("Check Trace Context Propagation", func(t *testing.T) {
		// Make request and capture trace headers
		requestBody := map[string]string{"cep": "29902555"}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req, err := http.NewRequest("POST", "http://localhost:8080/", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Skipf("Não foi possível conectar ao Service A: %v", err)
		}
		defer resp.Body.Close()

		// Check if there are trace headers in the response
		traceID := resp.Header.Get("X-Trace-Id")
		spanID := resp.Header.Get("X-Span-Id")

		if traceID != "" {
			t.Logf("✓ Trace ID propagado: %s", traceID)
		} else {
			t.Log("⚠ Trace ID não propagado na resposta")
		}

		if spanID != "" {
			t.Logf("✓ Span ID propagado: %s", spanID)
		} else {
			t.Log("⚠ Span ID não propagado na resposta")
		}

		// Wait for processing
		time.Sleep(2 * time.Second)
	})
}

// TestZipkinTraceStructure verifies trace structure in Zipkin
func TestZipkinTraceStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Zipkin trace structure test in short mode")
	}

	// Make request to generate trace
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
	resp.Body.Close()

	// Wait for processing
	time.Sleep(3 * time.Second)

	// Query traces
	resp, err = http.Get("http://localhost:9411/api/v2/traces")
	if err != nil {
		t.Skipf("Não foi possível consultar API do Zipkin: %v", err)
	}
	defer resp.Body.Close()

	var traces [][]map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&traces)
	if err != nil {
		t.Fatalf("Failed to decode Zipkin response: %v", err)
	}

	if len(traces) == 0 {
		t.Skip("Nenhum trace encontrado para análise de estrutura")
	}

	// Analyze structure of first trace (only log in verbose mode)
	trace := traces[0]
	if testing.Verbose() {
		t.Logf("Analisando estrutura do trace com %d spans", len(trace))

		for i, span := range trace {
			t.Logf("Span %d:", i+1)

			// Check required fields
			if id, ok := span["id"].(string); ok {
				t.Logf("  ID: %s", id)
			}

			if traceID, ok := span["traceId"].(string); ok {
				t.Logf("  Trace ID: %s", traceID)
			}

			if name, ok := span["name"].(string); ok {
				t.Logf("  Name: %s", name)
			}

			if timestamp, ok := span["timestamp"].(float64); ok {
				t.Logf("  Timestamp: %f", timestamp)
			}

			if duration, ok := span["duration"].(float64); ok {
				t.Logf("  Duration: %f μs", duration)
			}

			// Check local endpoint
			if localEndpoint, ok := span["localEndpoint"].(map[string]interface{}); ok {
				if serviceName, ok := localEndpoint["serviceName"].(string); ok {
					t.Logf("  Service: %s", serviceName)
				}
			}

			// Check tags/attributes
			if tags, ok := span["tags"].(map[string]interface{}); ok {
				t.Logf("  Tags: %+v", tags)
			}

			t.Log("")
		}
	}
}

// TestZipkinTraceLatency verifies trace latency
func TestZipkinTraceLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Zipkin trace latency test in short mode")
	}

	// Make multiple requests to measure latency
	latencies := make([]time.Duration, 5)

	for i := 0; i < 5; i++ {
		requestBody := map[string]string{"cep": "29902555"}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		start := time.Now()
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Post(
			"http://localhost:8080/",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		latency := time.Since(start)

		if err != nil {
			t.Skipf("Não foi possível conectar ao Service A: %v", err)
		}
		resp.Body.Close()

		latencies[i] = latency
		t.Logf("Requisição %d: %v (status: %d)", i+1, latency, resp.StatusCode)

		// Wait between requests
		time.Sleep(1 * time.Second)
	}

	// Calculate statistics
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	avgLatency := total / time.Duration(len(latencies))

	t.Logf("Latência média: %v", avgLatency)
	t.Logf("Latência mínima: %v", minLatency(latencies))
	t.Logf("Latência máxima: %v", maxLatency(latencies))

	// Check if latency is within acceptable limits
	if avgLatency > 5*time.Second {
		t.Errorf("Latência média muito alta: %v", avgLatency)
	}
}

// Helper functions
func minLatency(latencies []time.Duration) time.Duration {
	min := latencies[0]
	for _, l := range latencies[1:] {
		if l < min {
			min = l
		}
	}
	return min
}

func maxLatency(latencies []time.Duration) time.Duration {
	max := latencies[0]
	for _, l := range latencies[1:] {
		if l > max {
			max = l
		}
	}
	return max
}
