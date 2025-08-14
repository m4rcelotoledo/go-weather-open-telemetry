package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_WithEnvironmentVariables(t *testing.T) {
	// Set up test environment variables
	os.Setenv("WEATHER_API_KEY", "test-api-key-123")
	os.Setenv("PORT", "9090")
	defer func() {
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("PORT")
	}()

	// Load config
	cfg := LoadConfig()

	// Assertions
	assert.Equal(t, "test-api-key-123", cfg.WeatherAPIKey)
	assert.Equal(t, "9090", cfg.Port)
}

func TestLoadConfig_WithDefaultPort(t *testing.T) {
	// Set up test environment variables (only API key)
	os.Setenv("WEATHER_API_KEY", "test-api-key-456")
	os.Unsetenv("PORT") // Ensure PORT is not set
	defer func() {
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("PORT")
	}()

	// Load config
	cfg := LoadConfig()

	// Assertions
	assert.Equal(t, "test-api-key-456", cfg.WeatherAPIKey)
	assert.Equal(t, "8080", cfg.Port, "Should use default port 8080 when PORT is not set")
}

func TestLoadConfig_WithEmptyAPIKey(t *testing.T) {
	// Set up test environment variables
	os.Setenv("WEATHER_API_KEY", "")
	os.Setenv("PORT", "7070")
	defer func() {
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("PORT")
	}()

	// Load config
	cfg := LoadConfig()

	// Assertions
	assert.Equal(t, "", cfg.WeatherAPIKey)
	assert.Equal(t, "7070", cfg.Port)
}

func TestLoadConfig_WithNoEnvironmentVariables(t *testing.T) {
	// Ensure no environment variables are set
	os.Unsetenv("WEATHER_API_KEY")
	os.Unsetenv("PORT")
	defer func() {
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("PORT")
	}()

	// Load config
	cfg := LoadConfig()

	// Assertions
	assert.Equal(t, "", cfg.WeatherAPIKey)
	assert.Equal(t, "8080", cfg.Port, "Should use default port 8080 when no environment variables are set")
}

func TestLoadConfig_WithSpecialCharacters(t *testing.T) {
	// Set up test environment variables with special characters
	os.Setenv("WEATHER_API_KEY", "test-api-key-with-special-chars-!@#$%^&*()")
	os.Setenv("PORT", "12345")
	defer func() {
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("PORT")
	}()

	// Load config
	cfg := LoadConfig()

	// Assertions
	assert.Equal(t, "test-api-key-with-special-chars-!@#$%^&*()", cfg.WeatherAPIKey)
	assert.Equal(t, "12345", cfg.Port)
}

func TestLoadConfig_WithVeryLongAPIKey(t *testing.T) {
	// Create a very long API key (but not too long to avoid issues)
	longAPIKey := "very-long-api-key-" + strings.Repeat("a", 100)
	os.Setenv("WEATHER_API_KEY", longAPIKey)
	os.Setenv("PORT", "9999")
	defer func() {
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("PORT")
	}()

	// Load config
	cfg := LoadConfig()

	// Assertions
	assert.Equal(t, longAPIKey, cfg.WeatherAPIKey)
	assert.Equal(t, "9999", cfg.Port)
}

func TestLoadConfig_WithWhitespaceInValues(t *testing.T) {
	// Set up test environment variables with whitespace
	os.Setenv("WEATHER_API_KEY", "  test-api-key-with-whitespace  ")
	os.Setenv("PORT", "  8888  ")
	defer func() {
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("PORT")
	}()

	// Load config
	cfg := LoadConfig()

	// Assertions
	assert.Equal(t, "  test-api-key-with-whitespace  ", cfg.WeatherAPIKey)
	assert.Equal(t, "  8888  ", cfg.Port)
}

func TestLoadConfig_WithUnicodeCharacters(t *testing.T) {
	// Set up test environment variables with unicode characters
	os.Setenv("WEATHER_API_KEY", "test-api-key-中文-日本語-한국어")
	os.Setenv("PORT", "7777")
	defer func() {
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("PORT")
	}()

	// Load config
	cfg := LoadConfig()

	// Assertions
	assert.Equal(t, "test-api-key-中文-日本語-한국어", cfg.WeatherAPIKey)
	assert.Equal(t, "7777", cfg.Port)
}

func TestLoadConfig_Consistency(t *testing.T) {
	// Test that multiple calls return consistent results
	os.Setenv("WEATHER_API_KEY", "consistent-api-key")
	os.Setenv("PORT", "5555")
	defer func() {
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("PORT")
	}()

	// Load config multiple times
	cfg1 := LoadConfig()
	cfg2 := LoadConfig()
	cfg3 := LoadConfig()

	// Assertions - all should be identical
	assert.Equal(t, cfg1.WeatherAPIKey, cfg2.WeatherAPIKey)
	assert.Equal(t, cfg2.WeatherAPIKey, cfg3.WeatherAPIKey)
	assert.Equal(t, cfg1.Port, cfg2.Port)
	assert.Equal(t, cfg2.Port, cfg3.Port)
}
