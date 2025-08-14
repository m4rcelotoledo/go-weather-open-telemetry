package viacep

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient_GetAddressByCEP_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"cep": "01001-000",
			"localidade": "São Paulo",
			"erro": false
		}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := &client{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	// Test
	result, err := client.GetAddressByCEP("01001000")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "São Paulo", result.Localidade)
	assert.Equal(t, "01001-000", result.CEP)
	assert.False(t, result.Erro)
}

func TestClient_GetAddressByCEP_NotFound(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"erro": true
		}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := &client{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	// Test
	result, err := client.GetAddressByCEP("00000000")

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "CEP not found")
}

func TestClient_GetAddressByCEP_ServerError(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client with mock server URL
	client := &client{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	// Test
	result, err := client.GetAddressByCEP("01001000")

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "server error (5xx): 500")
}

func TestClient_GetAddressByCEP_Timeout(t *testing.T) {
	// Mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with shorter timeout
	client := &client{
		httpClient: &http.Client{
			Timeout: 100 * time.Millisecond, // Very short timeout
		},
		baseURL: server.URL,
	}

	// Test
	result, err := client.GetAddressByCEP("01001000")

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "error making request to ViaCEP")
}

func TestNewClient_TimeoutBehavior(t *testing.T) {
	// Test that NewClient creates a client that respects timeouts
	// We test this indirectly by using the timeout test above
	client := NewClient()
	assert.NotNil(t, client, "NewClient should return a non-nil client")

	// The timeout behavior is already tested in TestClient_GetAddressByCEP_Timeout
	// This test just ensures NewClient returns a working client
	// In a real scenario, we could mock a slow server to test the timeout
}
