package viacep

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Response struct {
	CEP        string `json:"cep"`
	Localidade string `json:"localidade"`
	Erro       bool   `json:"erro"`
}

type Client interface {
	GetAddressByCEP(cep string) (*Response, error)
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
		baseURL: "https://viacep.com.br/ws",
	}
}

func (c *client) GetAddressByCEP(cep string) (*Response, error) {
	url := fmt.Sprintf("%s/%s/json/", c.baseURL, cep)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request to ViaCEP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, fmt.Errorf("client error (4xx): %d", resp.StatusCode)
	} else if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server error (5xx): %d", resp.StatusCode)
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ViaCEP returned status code: %d", resp.StatusCode)
	}

	var viaCEPResp Response
	if err := json.NewDecoder(resp.Body).Decode(&viaCEPResp); err != nil {
		return nil, fmt.Errorf("error decoding ViaCEP response: %w", err)
	}

	if viaCEPResp.Erro {
		return nil, fmt.Errorf("CEP not found")
	}

	return &viaCEPResp, nil
}
