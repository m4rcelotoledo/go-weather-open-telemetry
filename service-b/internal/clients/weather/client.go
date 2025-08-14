package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Response struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

type Client interface {
	GetCurrentWeather(city, apiKey string) (*Response, error)
}

type client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	}

	return &client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
		baseURL: "https://api.weatherapi.com/v1",
	}
}

func (c *client) GetCurrentWeather(city, apiKey string) (*Response, error) {
	// Sanitize city name by removing accents
	sanitizedCity := c.sanitizeCityName(city)

	encodedCity := url.QueryEscape(sanitizedCity)
	requestURL := fmt.Sprintf("%s/current.json?key=%s&q=%s", c.baseURL, apiKey, encodedCity)

	resp, err := c.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("error making request to WeatherAPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// Read response body to get error details
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		errorMsg := string(body[:n])
		return nil, fmt.Errorf("client error (4xx): %d - %s", resp.StatusCode, errorMsg)
	} else if resp.StatusCode >= 500 {
		// Read response body to get error details
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		errorMsg := string(body[:n])
		return nil, fmt.Errorf("server error (5xx): %d - %s", resp.StatusCode, errorMsg)
	} else if resp.StatusCode != http.StatusOK {
		// Read response body to get error details
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		errorMsg := string(body[:n])
		return nil, fmt.Errorf("WeatherAPI returned status code %d: %s", resp.StatusCode, errorMsg)
	}

	var weatherResp Response
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		return nil, fmt.Errorf("error decoding WeatherAPI response: %w", err)
	}

	return &weatherResp, nil
}

// sanitizeCityName removes accents and special characters from city name
func (c *client) sanitizeCityName(city string) string {
	// Map of accented character replacements
	replacements := map[string]string{
		"á": "a", "à": "a", "ã": "a", "â": "a", "ä": "a",
		"é": "e", "è": "e", "ê": "e", "ë": "e",
		"í": "i", "ì": "i", "î": "i", "ï": "i",
		"ó": "o", "ò": "o", "õ": "o", "ô": "o", "ö": "o",
		"ú": "u", "ù": "u", "û": "u", "ü": "u",
		"ç": "c",
		"ñ": "n",
		"Á": "A", "À": "A", "Ã": "A", "Â": "A", "Ä": "A",
		"É": "E", "È": "E", "Ê": "E", "Ë": "E",
		"Í": "I", "Ì": "I", "Î": "I", "Ï": "I",
		"Ó": "O", "Ò": "O", "Õ": "O", "Ô": "O", "Ö": "O",
		"Ú": "U", "Ù": "U", "Û": "U", "Ü": "U",
		"Ç": "C",
		"Ñ": "N",
	}

	result := city
	for accented, plain := range replacements {
		result = strings.ReplaceAll(result, accented, plain)
	}

	return result
}
