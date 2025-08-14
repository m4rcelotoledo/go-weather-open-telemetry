package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"service-b/internal/clients/viacep"
	"service-b/internal/clients/weather"
	"service-b/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock clients
type MockViaCEPClient struct {
	mock.Mock
}

func (m *MockViaCEPClient) GetAddressByCEP(cep string) (*viacep.Response, error) {
	args := m.Called(cep)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*viacep.Response), args.Error(1)
}

type MockWeatherClient struct {
	mock.Mock
}

func (m *MockWeatherClient) GetCurrentWeather(city, apiKey string) (*weather.Response, error) {
	args := m.Called(city, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*weather.Response), args.Error(1)
}

func TestOrchestratorHandler_HandleWeatherRequest_Success(t *testing.T) {
	// Setup
	mockViaCEP := new(MockViaCEPClient)
	mockWeather := new(MockWeatherClient)
	cfg := &config.Config{
		WeatherAPIKey: "test-api-key",
		Port:          "8081",
	}
	handler := NewOrchestratorHandler(mockViaCEP, mockWeather, cfg)

	// Mock responses
	viaCEPResponse := &viacep.Response{
		CEP:        "29902-555",
		Localidade: "Vila Velha",
		Erro:       false,
	}
	weatherResponse := &weather.Response{
		Current: struct {
			TempC float64 `json:"temp_c"`
		}{TempC: 25.5},
	}

	mockViaCEP.On("GetAddressByCEP", "29902555").Return(viaCEPResponse, nil)
	mockWeather.On("GetCurrentWeather", "Vila Velha", "test-api-key").Return(weatherResponse, nil)

	// Create request
	requestBody := CEPRequest{CEP: "29902555"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/weather", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.HandleWeatherRequest(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response WeatherResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Vila Velha", response.City)
	assert.Equal(t, 25.5, response.TempC)
	assert.Equal(t, 77.9, response.TempF)
	assert.Equal(t, 298.6, response.TempK)

	mockViaCEP.AssertExpectations(t)
	mockWeather.AssertExpectations(t)
}

func TestOrchestratorHandler_HandleWeatherRequest_InvalidMethod(t *testing.T) {
	// Setup
	mockViaCEP := new(MockViaCEPClient)
	mockWeather := new(MockWeatherClient)
	cfg := &config.Config{
		WeatherAPIKey: "test-api-key",
		Port:          "8081",
	}
	handler := NewOrchestratorHandler(mockViaCEP, mockWeather, cfg)

	// Create request with GET method
	req := httptest.NewRequest("GET", "/weather", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.HandleWeatherRequest(w, req)

	// Assertions
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestOrchestratorHandler_HandleWeatherRequest_InvalidJSON(t *testing.T) {
	// Setup
	mockViaCEP := new(MockViaCEPClient)
	mockWeather := new(MockWeatherClient)
	cfg := &config.Config{
		WeatherAPIKey: "test-api-key",
		Port:          "8081",
	}
	handler := NewOrchestratorHandler(mockViaCEP, mockWeather, cfg)

	// Create request with invalid JSON
	req := httptest.NewRequest("POST", "/weather", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.HandleWeatherRequest(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestOrchestratorHandler_HandleWeatherRequest_InvalidCEP(t *testing.T) {
	// Setup
	mockViaCEP := new(MockViaCEPClient)
	mockWeather := new(MockWeatherClient)
	cfg := &config.Config{
		WeatherAPIKey: "test-api-key",
		Port:          "8081",
	}
	handler := NewOrchestratorHandler(mockViaCEP, mockWeather, cfg)

	// Create request with invalid CEP
	requestBody := CEPRequest{CEP: "123"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/weather", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.HandleWeatherRequest(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestOrchestratorHandler_HandleWeatherRequest_CEPNotFound(t *testing.T) {
	// Setup
	mockViaCEP := new(MockViaCEPClient)
	mockWeather := new(MockWeatherClient)
	cfg := &config.Config{
		WeatherAPIKey: "test-api-key",
		Port:          "8081",
	}
	handler := NewOrchestratorHandler(mockViaCEP, mockWeather, cfg)

	// Mock ViaCEP returning error
	mockViaCEP.On("GetAddressByCEP", "00000000").Return(nil, assert.AnError)

	// Create request
	requestBody := CEPRequest{CEP: "00000000"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/weather", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.HandleWeatherRequest(w, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	mockViaCEP.AssertExpectations(t)
}

func TestOrchestratorHandler_HandleWeatherRequest_WeatherAPIError(t *testing.T) {
	// Setup
	mockViaCEP := new(MockViaCEPClient)
	mockWeather := new(MockWeatherClient)
	cfg := &config.Config{
		WeatherAPIKey: "test-api-key",
		Port:          "8081",
	}
	handler := NewOrchestratorHandler(mockViaCEP, mockWeather, cfg)

	// Mock ViaCEP success but WeatherAPI error
	viaCEPResponse := &viacep.Response{
		CEP:        "29902-555",
		Localidade: "Vila Velha",
		Erro:       false,
	}

	mockViaCEP.On("GetAddressByCEP", "29902555").Return(viaCEPResponse, nil)
	mockWeather.On("GetCurrentWeather", "Vila Velha", "test-api-key").Return(nil, assert.AnError)

	// Create request
	requestBody := CEPRequest{CEP: "29902555"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/weather", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.HandleWeatherRequest(w, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	mockViaCEP.AssertExpectations(t)
	mockWeather.AssertExpectations(t)
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
		{"Invalid CEP too short", "1234567", false},
		{"Invalid CEP too long", "123456789", false},
		{"Invalid CEP with letters", "abc12345", false},
		{"Empty CEP", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidCEP(tt.cep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRoundToOneDecimal(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"Integer", 25.0, 25.0},
		{"One decimal", 25.5, 25.5},
		{"Multiple decimals", 25.567, 25.6},
		{"Negative", -10.3, -10.3},
		{"Zero", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roundToOneDecimal(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewOrchestratorHandler(t *testing.T) {
	// Setup
	mockViaCEP := new(MockViaCEPClient)
	mockWeather := new(MockWeatherClient)
	cfg := &config.Config{
		WeatherAPIKey: "test-api-key",
		Port:          "8081",
	}

	// Execute
	handler := NewOrchestratorHandler(mockViaCEP, mockWeather, cfg)

	// Assertions
	assert.NotNil(t, handler)
	assert.Equal(t, mockViaCEP, handler.viaCEPClient)
	assert.Equal(t, mockWeather, handler.weatherClient)
	assert.Equal(t, cfg, handler.config)
}
