package clients

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func Test_WeatherStack_QuerySuccessful(t *testing.T) {
	client, roundTripper := initializeWeatherStackTestClient(t)
	expectedTemp := 20.5
	roundTripper.AddResponse(getWeatherStackSuccessfulResponse(t, expectedTemp))
	actualTemp, err := client.Query(context.Background(), "Istanbul")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if actualTemp != expectedTemp {
		t.Fatalf("expected %v, got %v", expectedTemp, actualTemp)
	}
	validateWeatherStackRequest(t, roundTripper.GetLastRequest())
}

func Test_WeatherStack_QueryFailed(t *testing.T) {
	client, roundTripper := initializeWeatherStackTestClient(t)
	roundTripper.AddResponse(getWeatherStackErrorResponse(t))
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
	validateWeatherStackRequest(t, roundTripper.GetLastRequest())
}

func initializeWeatherStackTestClient(t *testing.T) (*WeatherStackClient, *MockRoundTripper) {
	t.Helper()
	roundTripper := NewMockRoundTripper()
	return &WeatherStackClient{
		baseUrl:   "http://test-url",
		apiKey:    "test-api-key",
		transport: &http.Client{Transport: roundTripper},
	}, roundTripper
}

func getWeatherStackSuccessfulResponse(t *testing.T, temperature float64) *http.Response {
	t.Helper()
	body := fmt.Sprintf(`{"success": true, "current": {"temperature": %v}}`, temperature)
	return &http.Response{
		StatusCode:    http.StatusOK,
		Body:          io.NopCloser(strings.NewReader(body)),
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		ContentLength: int64(len(body)),
	}
}

func getWeatherStackErrorResponse(t *testing.T) *http.Response {
	t.Helper()
	body := `{"success": false, "error": {"code": 401, "type": "usage_limit_reached", "info": "Your monthly usage limit has been reached."}}`
	return &http.Response{
		StatusCode:    http.StatusUnauthorized,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

func validateWeatherStackRequest(t *testing.T, req *http.Request) {
	if req == nil {
		t.Fatal("expected request to be not nil")
	}
	if req.Method != http.MethodGet {
		t.Fatalf("expected method %s, got %s", http.MethodGet, req.Method)
	}
	expectedURL := "http://test-url/current?access_key=test-api-key&query=Istanbul"
	if req.URL.String() != expectedURL {
		t.Fatalf("expected url %s, got %s", expectedURL, req.URL.String())
	}
}
