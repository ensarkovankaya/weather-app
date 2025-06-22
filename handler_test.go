package main

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockClient struct {
	Name string
	Err  error
	Temp float64
}

func (c *mockClient) Query(_ context.Context, location string) (float64, error) {
	sleepDuration := time.Duration(rand.Intn(1000)) * time.Millisecond // simulate random API response delay
	log.Println(fmt.Sprintf("Mock client simulating API response delay for location: %s, sleep duration: %v, client: %v", location, sleepDuration, c.Name))
	time.Sleep(sleepDuration)
	return c.Temp, c.Err
}

type testCase struct {
	Name            string
	Location        string
	Client1         *mockClient
	Client2         *mockClient
	ExpectedTemp    float64
	RequestCount    int
	ResponseTimeout int
}

func TestSuccessfulResponse(t *testing.T) {
	cases := []testCase{
		{
			Name:            "Single Request",
			Location:        "Istanbul",
			Client1:         &mockClient{Name: "Client1", Temp: 20.0},
			Client2:         &mockClient{Name: "Client2", Temp: 22.0},
			ExpectedTemp:    21.0,
			RequestCount:    1,
			ResponseTimeout: int(6 * time.Second.Milliseconds()),
		},
		{
			Name:            "Less than 10 requests",
			Location:        "Istanbul",
			Client1:         &mockClient{Name: "Client1", Temp: 16.0},
			Client2:         &mockClient{Name: "Client2", Temp: 17.0},
			ExpectedTemp:    16.5,
			RequestCount:    rand.Intn(10) + 2, // Random number of requests between 2 and 10
			ResponseTimeout: int(6 * time.Second.Milliseconds()),
		},
		{
			Name:            "Exactly 10 requests",
			Location:        "Istanbul",
			Client1:         &mockClient{Name: "Client1", Temp: 20.0},
			Client2:         &mockClient{Name: "Client2", Temp: 22.0},
			ExpectedTemp:    21.0,
			RequestCount:    10,
			ResponseTimeout: int(5 * time.Second.Milliseconds()), // should take less than 5 seconds
		},
		{
			Name:            "More than 10 requests",
			Location:        "Istanbul",
			Client1:         &mockClient{Name: "Client1", Temp: 20.0},
			Client2:         &mockClient{Name: "Client2", Temp: 22.0},
			ExpectedTemp:    21.0,
			RequestCount:    rand.Intn(10) + 10,
			ResponseTimeout: int(5 * time.Second.Milliseconds()), // should take less than 5 seconds
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testSuccessfulResponse(t, tc)
		})
	}
}

func testSuccessfulResponse(t *testing.T, tc testCase) {
	// Mock the database
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// Mock the expected database insert operation for every 10 requests
	remainingRequests := tc.RequestCount
	requestCount := tc.RequestCount
	for remainingRequests > 0 {
		if remainingRequests > 10 {
			requestCount = 10
			remainingRequests -= 10
		} else {
			requestCount = remainingRequests
			remainingRequests = 0
		}
		mock.ExpectExec(`INSERT INTO weather_queries (location, service_1_temperature, service_2_temperature, request_count, created_at) VALUES (?, ?, ?, ?, datetime('now'))`).
			WithArgs(tc.Location, tc.Client1.Temp, tc.Client2.Temp, requestCount).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	// Create the handler with mocked clients
	handler := NewWeatherHandler(db, tc.Client1, tc.Client2)

	// initialize the Fiber app
	app := fiber.New()
	handler.Setup(app)

	// Make parallel requests
	wg := &sync.WaitGroup{}
	lock := &sync.Mutex{}
	responses := make([]*http.Response, 0, tc.RequestCount)
	wg.Add(tc.RequestCount)
	for i := 0; i < tc.RequestCount; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond) // simulate random request delay
			req := httptest.NewRequest(http.MethodGet, "/weather?q="+tc.Location, nil)
			resp, err := app.Test(req, tc.ResponseTimeout)
			if err != nil {
				t.Errorf("Failed to create test request: %v", err)
				return
			}
			lock.Lock()
			defer lock.Unlock()
			responses = append(responses, resp)
		}()
	}
	wg.Wait()

	// Validate responses
	for _, resp := range responses {
		validateSuccessfulResponse(t, resp, tc.Location, tc.ExpectedTemp)
	}

	// Check if db mock expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("There were unfulfilled expectations: %s", err)
	}
}

func validateSuccessfulResponse(t *testing.T, resp *http.Response, location string, expectedTemp float64) {
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	expectedResponse := fmt.Sprintf(`{"location":"%v","temperature":%v}`, location, expectedTemp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != expectedResponse {
		t.Fatalf("Expected response body '%s', got '%s'", expectedResponse, string(body))
	}
}
