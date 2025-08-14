package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"service-b/internal/clients/viacep"
	"service-b/internal/clients/weather"
	"service-b/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type OrchestratorHandler struct {
	viaCEPClient  viacep.Client
	weatherClient weather.Client
	config        *config.Config
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

func NewOrchestratorHandler(viaCEPClient viacep.Client, weatherClient weather.Client, cfg *config.Config) *OrchestratorHandler {
	return &OrchestratorHandler{
		viaCEPClient:  viaCEPClient,
		weatherClient: weatherClient,
		config:        cfg,
	}
}

func (h *OrchestratorHandler) HandleWeatherRequest(w http.ResponseWriter, r *http.Request) {
	// Get tracer
	tracer := otel.Tracer("service-b")
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

	// Find city by CEP
	address, err := h.getCityByCEP(ctx, cepReq.CEP)
	if err != nil {
		log.Printf("CEP not found: %s - Error: %v", cepReq.CEP, err)
		span.SetAttributes(attribute.String("error", "cep_not_found"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "can not find zipcode"})
		return
	}

	// Log found city
	log.Printf("Found city: %s for CEP: %s", address.Localidade, cepReq.CEP)
	span.SetAttributes(attribute.String("city", address.Localidade))

	// Get current weather
	weatherData, err := h.getWeatherData(ctx, address.Localidade)
	if err != nil {
		log.Printf("Error fetching weather data for %s: %v", address.Localidade, err)
		span.SetAttributes(attribute.String("error", "weather_api_error"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Error fetching weather data"})
		return
	}

	// Calculate temperatures
	tempC := weatherData.Current.TempC
	tempF := tempC*1.8 + 32
	tempK := tempC + 273.15

	// Log temperature data
	log.Printf("Temperature retrieved: %.1f°C for %s", tempC, address.Localidade)
	span.SetAttributes(attribute.Float64("temp_c", tempC))

	response := WeatherResponse{
		City:  address.Localidade,
		TempC: roundToOneDecimal(tempC),
		TempF: roundToOneDecimal(tempF),
		TempK: roundToOneDecimal(tempK),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *OrchestratorHandler) getCityByCEP(ctx context.Context, cep string) (*viacep.Response, error) {
	tracer := otel.Tracer("service-b")
	ctx, span := tracer.Start(ctx, "request-viacep")
	defer span.End()

	span.SetAttributes(attribute.String("cep", cep))
	span.SetAttributes(attribute.String("api_url", "https://viacep.com.br"))

	// Call ViaCEP API
	address, err := h.viaCEPClient.GetAddressByCEP(cep)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, err
	}

	// Check if CEP was found
	if address.Erro {
		span.SetAttributes(attribute.String("error", "cep_not_found"))
		return nil, fmt.Errorf("CEP not found")
	}

	span.SetAttributes(attribute.String("city", address.Localidade))
	return address, nil
}

func (h *OrchestratorHandler) getWeatherData(ctx context.Context, city string) (*weather.Response, error) {
	tracer := otel.Tracer("service-b")
	ctx, span := tracer.Start(ctx, "request-weatherapi")
	defer span.End()

	span.SetAttributes(attribute.String("city", city))
	span.SetAttributes(attribute.String("api_url", "http://api.weatherapi.com"))

	// Check if API key is configured
	if h.config.WeatherAPIKey == "" {
		span.SetAttributes(attribute.String("error", "api_key_not_configured"))
		return nil, fmt.Errorf("Weather API key not configured")
	}

	// Get current weather
	weatherData, err := h.weatherClient.GetCurrentWeather(city, h.config.WeatherAPIKey)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, err
	}

	span.SetAttributes(attribute.Float64("temp_c", weatherData.Current.TempC))
	return weatherData, nil
}

func isValidCEP(cep string) bool {
	// Remove spaces and hyphens
	cleaned := strings.ReplaceAll(strings.ReplaceAll(cep, "-", ""), " ", "")
	matched, _ := regexp.MatchString(`^\d{8}$`, cleaned)
	return matched
}

func roundToOneDecimal(value float64) float64 {
	rounded, _ := strconv.ParseFloat(strconv.FormatFloat(value, 'f', 1, 64), 64)
	return rounded
}
