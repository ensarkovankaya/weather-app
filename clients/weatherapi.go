package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/ensarkovankaya/weather-app/settings"
)

type WeatherAPIResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	}
}

// WeatherAPIClient is a client for querying weather data from the WeatherAPI service.
type WeatherAPIClient struct {
	url       string
	apiKey    string
	transport *http.Client
}

// NewWeatherAPIClient creates a new instance of WeatherAPIClient with the configuration from settings.
func NewWeatherAPIClient() *WeatherAPIClient {
	return &WeatherAPIClient{
		url:       settings.Cnf.WeatherAPIURL,
		apiKey:    settings.Cnf.WeatherAPIKey,
		transport: http.DefaultClient,
	}
}

// Query fetches the current temperature for a given location from the WeatherAPI service.
func (w *WeatherAPIClient) Query(ctx context.Context, location string) (float64, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/current.json?key=%s&q=%s&aqi=no", w.url, w.apiKey, url.QueryEscape(location)),
		nil,
	)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %w", err)
	}
	resp, err := w.transport.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data := &WeatherAPIResponse{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return 0, fmt.Errorf("error decoding response: %w", err)
	}

	log.Println("Temperature from WeatherAPI:", data.Current.TempC)
	return data.Current.TempC, nil
}
