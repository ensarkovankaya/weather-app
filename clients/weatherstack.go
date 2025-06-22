package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"weather-app/settings"
)

type WeatherStackResponse struct {
	Success bool `json:"success"`
	Current struct {
		Temperature float64 `json:"temperature"`
	}
	Error struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	}
}

// WeatherStackClient is a client for querying weather data from the WeatherStack service.
type WeatherStackClient struct {
	baseUrl   string
	apiKey    string
	transport *http.Client
}

// NewWeatherStackClient creates a new instance of WeatherStackClient with the configuration from settings.
func NewWeatherStackClient() *WeatherStackClient {
	return &WeatherStackClient{
		baseUrl:   settings.Cnf.WeatherStackAPIURL,
		apiKey:    settings.Cnf.WeatherStackAPIKey,
		transport: http.DefaultClient,
	}
}

func (c *WeatherStackClient) Query(ctx context.Context, location string) (float64, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/current?access_key=%s&query=%s", c.baseUrl, c.apiKey, url.QueryEscape(location)),
		nil,
	)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.transport.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to query weather stack: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data := &WeatherStackResponse{}
	if err = json.NewDecoder(resp.Body).Decode(data); err != nil {
		return 0, fmt.Errorf("error decoding response: %w", err)
	}

	if !data.Success {
		return 0, fmt.Errorf("weather stack API error: 'success' field is false")
	}

	log.Println("Temperature from WeatherStack:", data.Current.Temperature)
	return data.Current.Temperature, nil

}
