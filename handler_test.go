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
	Name               string
	Location           string
	Client1            *mockClient
	Client2            *mockClient
	RequestCount       int
	ResponseTimeout    int
	ResponseStatusCode int
	ResponseBody       string
}

func TestLocationRequiredResponse(t *testing.T) {
	runTestCase(t, testCase{
		RequestCount:       1,
		ResponseTimeout:    int(2 * time.Second.Milliseconds()),
		ResponseStatusCode: http.StatusBadRequest,
		ResponseBody:       `{"error":"Location query parameter is required"}`,
	})
}

func TestSuccessfulResponse(t *testing.T) {
	cases := []testCase{
		{
			Name:               "Single Request",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Temp: 20.0},
			Client2:            &mockClient{Name: "Client2", Temp: 22.0},
			RequestCount:       1,
			ResponseTimeout:    int(6 * time.Second.Milliseconds()), // 5 seconds for aggregating requests ~1 second for processing
			ResponseStatusCode: http.StatusOK,
			ResponseBody:       `{"location":"Istanbul","temperature":21}`,
		},
		{
			Name:               "Less than 10 requests",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Temp: 16.0},
			Client2:            &mockClient{Name: "Client2", Temp: 17.0},
			RequestCount:       rand.Intn(10) + 2,                   // Random number of requests between 2 and 10
			ResponseTimeout:    int(6 * time.Second.Milliseconds()), // 5 seconds for aggregating requests ~1 second for processing
			ResponseStatusCode: http.StatusOK,
			ResponseBody:       `{"location":"Istanbul","temperature":16.5}`,
		},
		{
			Name:               "10 requests",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Temp: 20.0},
			Client2:            &mockClient{Name: "Client2", Temp: 22.0},
			RequestCount:       10,
			ResponseTimeout:    int(5 * time.Second.Milliseconds()), // should take less than 5 seconds
			ResponseStatusCode: http.StatusOK,
			ResponseBody:       `{"location":"Istanbul","temperature":21}`,
		},
		{
			Name:               "More than 10 requests",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Temp: 20.0},
			Client2:            &mockClient{Name: "Client2", Temp: 22.0},
			RequestCount:       rand.Intn(10) + 10,
			ResponseTimeout:    int(7 * time.Second.Milliseconds()), // should take less than 7 seconds
			ResponseStatusCode: http.StatusOK,
			ResponseBody:       `{"location":"Istanbul","temperature":21}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

func TestSuccessfulResponseIfOneOfTheClientFail(t *testing.T) {
	cases := []testCase{
		{
			Name:               "Single Request",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Temp: 20.0},
			Client2:            &mockClient{Name: "Client2", Err: fmt.Errorf("test error")},
			RequestCount:       1,
			ResponseTimeout:    int(6 * time.Second.Milliseconds()), // 5 seconds for aggregating requests ~1 second for processing
			ResponseStatusCode: http.StatusOK,
			ResponseBody:       `{"location":"Istanbul","temperature":20}`,
		},
		{
			Name:               "Less than 10 requests",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Err: fmt.Errorf("test error")},
			Client2:            &mockClient{Name: "Client2", Temp: 16.0},
			RequestCount:       rand.Intn(10) + 2,                   // Random number of requests between 2 and 10
			ResponseTimeout:    int(6 * time.Second.Milliseconds()), // 5 seconds for aggregating requests ~1 second for processing
			ResponseStatusCode: http.StatusOK,
			ResponseBody:       `{"location":"Istanbul","temperature":16}`,
		},
		{
			Name:               "10 requests",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Temp: 20.0},
			Client2:            &mockClient{Name: "Client2", Err: fmt.Errorf("test error")},
			RequestCount:       10,
			ResponseTimeout:    int(5 * time.Second.Milliseconds()), // should take less than 5 seconds
			ResponseStatusCode: http.StatusOK,
			ResponseBody:       `{"location":"Istanbul","temperature":20}`,
		},
		{
			Name:               "More than 10 requests",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Err: fmt.Errorf("test error")},
			Client2:            &mockClient{Name: "Client2", Temp: 22.0},
			RequestCount:       rand.Intn(10) + 10,
			ResponseTimeout:    int(7 * time.Second.Milliseconds()), // should take less than 7 seconds
			ResponseStatusCode: http.StatusOK,
			ResponseBody:       `{"location":"Istanbul","temperature":22}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

func TestFailedResponseIfAllTheClientsFail(t *testing.T) {
	cases := []testCase{
		{
			Name:               "Single Request",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Err: fmt.Errorf("test error")},
			Client2:            &mockClient{Name: "Client2", Err: fmt.Errorf("test error")},
			RequestCount:       1,
			ResponseTimeout:    int(6 * time.Second.Milliseconds()), // 5 seconds for aggregating requests ~1 second for processing
			ResponseStatusCode: http.StatusInternalServerError,
			ResponseBody:       `{"error":"Failed to fetch weather data"}`,
		},
		{
			Name:               "Less than 10 requests",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Err: fmt.Errorf("test error")},
			Client2:            &mockClient{Name: "Client2", Err: fmt.Errorf("test error")},
			RequestCount:       rand.Intn(10) + 2,                   // Random number of requests between 2 and 10
			ResponseTimeout:    int(6 * time.Second.Milliseconds()), // 5 seconds for aggregating requests ~1 second for processing
			ResponseStatusCode: http.StatusInternalServerError,
			ResponseBody:       `{"error":"Failed to fetch weather data"}`,
		},
		{
			Name:               "10 requests",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Err: fmt.Errorf("test error")},
			Client2:            &mockClient{Name: "Client2", Err: fmt.Errorf("test error")},
			RequestCount:       10,
			ResponseTimeout:    int(5 * time.Second.Milliseconds()), // should take less than 5 seconds
			ResponseStatusCode: http.StatusInternalServerError,
			ResponseBody:       `{"error":"Failed to fetch weather data"}`,
		},
		{
			Name:               "More than 10 requests",
			Location:           "Istanbul",
			Client1:            &mockClient{Name: "Client1", Err: fmt.Errorf("test error")},
			Client2:            &mockClient{Name: "Client2", Err: fmt.Errorf("test error")},
			RequestCount:       rand.Intn(10) + 10,
			ResponseTimeout:    int(7 * time.Second.Milliseconds()), // should take less than 7 seconds
			ResponseStatusCode: http.StatusInternalServerError,
			ResponseBody:       `{"error":"Failed to fetch weather data"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

func runTestCase(t *testing.T, tc testCase) {
	log.Printf("Running test case: %+v\n", tc)
	// Mock the database
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// Mock the expected database insert operation for every 10 requests
	remainingRequests := tc.RequestCount
	requestCount := tc.RequestCount
	for remainingRequests > 0 && tc.ResponseStatusCode == http.StatusOK {
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
		validateResponse(t, resp, tc)
	}

	// Allow some time for logging operations to complete, especially when handling more than 10 requests.
	time.Sleep(500 * time.Millisecond)
	// Check if db mock expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("There were unfulfilled expectations: %s", err)
	}
}

func validateResponse(t *testing.T, resp *http.Response, tc testCase) {
	if resp.StatusCode != tc.ResponseStatusCode {
		t.Fatalf("Expected status code %d, got %d", tc.ResponseStatusCode, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != tc.ResponseBody {
		t.Fatalf("Expected response body '%s', got '%s'", tc.ResponseBody, string(body))
	}
}
