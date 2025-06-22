package clients

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func Test_WeatherAPI_QuerySuccessful(t *testing.T) {
	client, roundTripper := initializeWeatherAPITestClient(t)
	expectedTemp := 20.5
	roundTripper.AddResponse(getWeatherAPISuccessfulResponse(t, expectedTemp))
	actualTemp, err := client.Query(context.Background(), "Istanbul")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if actualTemp != expectedTemp {
		t.Fatalf("expected %v, got %v", expectedTemp, actualTemp)
	}
	validateWeatherAPIRequest(t, roundTripper.GetLastRequest())
}

func Test_WeatherAPI_QueryFailed(t *testing.T) {
	client, roundTripper := initializeWeatherAPITestClient(t)
	roundTripper.AddResponse(getWeatherAPIErrorResponse(t))
	actualTemp, err := client.Query(context.Background(), "Istanbul")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if actualTemp != 0 {
		t.Fatalf("expected temperature 0, got %v", actualTemp)
	}
	if !strings.Contains(err.Error(), "unexpected status code") {
		t.Fatalf("expected error to contain 'unexpected status code', got %v", err)
	}
	validateWeatherAPIRequest(t, roundTripper.GetLastRequest())
}

func initializeWeatherAPITestClient(t *testing.T) (*WeatherAPIClient, *MockRoundTripper) {
	t.Helper()
	roundTripper := NewMockRoundTripper()
	return &WeatherAPIClient{
		url:       "http://test-url",
		apiKey:    "test-api-key",
		transport: &http.Client{Transport: roundTripper},
	}, roundTripper
}

func getWeatherAPISuccessfulResponse(t *testing.T, temperature float64) *http.Response {
	t.Helper()
	body := fmt.Sprintf(`{"current": {"temp_c": %v}}`, temperature)
	return &http.Response{
		StatusCode:    http.StatusOK,
		Body:          io.NopCloser(strings.NewReader(body)),
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		ContentLength: int64(len(body)),
	}
}

func getWeatherAPIErrorResponse(t *testing.T) *http.Response {
	t.Helper()
	body := `{"error": {"code": 401, "message": "Unauthorized"}}`
	return &http.Response{
		StatusCode:    http.StatusUnauthorized,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

func validateWeatherAPIRequest(t *testing.T, req *http.Request) {
	if req == nil {
		t.Fatal("expected request to be not nil")
	}
	if req.Method != http.MethodGet {
		t.Fatalf("expected method %s, got %s", http.MethodGet, req.Method)
	}
	expectedURL := "http://test-url/current.json?key=test-api-key&q=Istanbul&aqi=no"
	if req.URL.String() != expectedURL {
		t.Fatalf("expected url %s, got %s", expectedURL, req.URL.String())
	}
}
