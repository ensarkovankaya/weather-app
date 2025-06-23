package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"log"
	"sync"
	"time"
)

var failedTemperature = -999.0 // Used to indicate a failed temperature fetch

type Client interface {
	Query(ctx context.Context, location string) (float64, error)
}

// WeatherResponse defines the JSON response structure
type WeatherResponse struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
}

type WeatherHandler struct {
	db      *sql.DB
	client1 Client
	client2 Client
	queries map[string]chan chan WeatherResponse
	lock    sync.Locker
}

func NewWeatherHandler(db *sql.DB, client1, client2 Client) *WeatherHandler {
	return &WeatherHandler{
		db:      db,
		client1: client1,
		client2: client2,
		queries: make(map[string]chan chan WeatherResponse),
		lock:    &sync.Mutex{},
	}
}

func (h *WeatherHandler) Setup(router fiber.Router) {
	router.Get("/weather/", h.Query)
}

// Query handles the weather query requests
func (h *WeatherHandler) Query(c *fiber.Ctx) error {
	location := c.Query("q")
	if location == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Location query parameter is required",
		})
	}

	resultChan := h.getChannel(c.UserContext(), location)

	// Wait for response and send it to the client
	response := <-resultChan
	if response.Temperature == failedTemperature {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch weather data"})
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *WeatherHandler) queryLocation(ctx context.Context, location string, weatherChannel chan chan WeatherResponse) {
	resultChannels := make([]chan WeatherResponse, 0)
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	var stop bool

	// Collect requests up to 5 seconds or 10 requests
	for !stop {
		select {
		case rc := <-weatherChannel:
			resultChannels = append(resultChannels, rc) // Add the channel to the list of result channels
			log.Println("Received request for location:", location, len(resultChannels))
			if len(resultChannels) >= 10 {
				stop = true // Stop collecting after 10 requests
			}
		case <-timer.C:
			stop = true // Stop collecting after 5 seconds
		}
	}
	h.cleanupChannel(location)

	// Query the weather API concurrently
	var (
		temp1, temp2 float64
		err1, err2   error
		wg           = &sync.WaitGroup{}
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		if temp1, err1 = h.client1.Query(ctx, location); err1 != nil {
			log.Printf("Error fetching temperature from WeatherAPIClient: %v", err1)
		}
	}()
	go func() {
		defer wg.Done()
		if temp2, err2 = h.client2.Query(ctx, location); err2 != nil {
			log.Printf("Error fetching temperature from WeatherAPIClient: %v", err2)
		}
	}()
	wg.Wait()

	// Calculate average temperature and prepare response
	var avgTemp float64
	if err1 != nil && err2 != nil {
		log.Println("Both API calls failed, returning failed temperature")
		avgTemp = failedTemperature
	} else if err1 == nil && err2 != nil {
		log.Println("Only first API call succeeded, using its temperature")
		avgTemp = temp1
	} else if err1 != nil && err2 == nil {
		log.Println("Only second API call succeeded, using its temperature")
		avgTemp = temp2
	} else {
		log.Println("Both API calls succeeded, calculating average temperature")
		avgTemp = (temp1 + temp2) / 2
	}
	response := WeatherResponse{
		Location:    location,
		Temperature: avgTemp,
	}

	// Send response to all waiting requests
	for _, rc := range resultChannels {
		rc <- response
		close(rc)
	}

	// Log the request to the database
	h.logToDatabase(location, temp1, temp2, len(resultChannels))
}

func (h *WeatherHandler) logToDatabase(location string, temp1, temp2 float64, requestCount int) {
	log.Println(fmt.Sprintf("Logging to database: location=%s, temp1=%.2f, temp2=%.2f, requestCount=%d", location, temp1, temp2, requestCount))
	_, err := h.db.Exec("INSERT INTO weather_queries (location, service_1_temperature, service_2_temperature, request_count, created_at) VALUES (?, ?, ?, ?, datetime('now'))",
		location, temp1, temp2, requestCount)
	if err != nil {
		log.Println("[Error] Failed to log to database:", err)
	}
}

func (h *WeatherHandler) cleanupChannel(location string) {
	h.lock.Lock()
	defer h.lock.Unlock()
	delete(h.queries, location)
}

func (h *WeatherHandler) getChannel(ctx context.Context, location string) chan WeatherResponse {
	// Check if there's an active query for this location
	// If not, create a new channel and start the query
	h.lock.Lock()
	defer h.lock.Unlock()
	weatherChannel, exists := h.queries[location]
	if !exists {
		weatherChannel = make(chan chan WeatherResponse, 10) // Limit to 10 concurrent requests
		h.queries[location] = weatherChannel
		go h.queryLocation(ctx, location, weatherChannel)
	}
	resultChan := make(chan WeatherResponse)
	weatherChannel <- resultChan
	return resultChan
}
