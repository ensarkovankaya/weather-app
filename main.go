package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ensarkovankaya/weather-app/clients"
	"github.com/ensarkovankaya/weather-app/settings"
)

func main() {
	db, err := initializeDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println("Failed to close database:", err)
		}
	}()

	// Clients
	client1 := clients.NewWeatherAPIClient()
	client2 := clients.NewWeatherStackClient()

	// Handlers
	weatherHandler := NewWeatherHandler(db, client1, client2)

	app := fiber.New()
	weatherHandler.Setup(app)

	go func() {
		address := fmt.Sprintf(":%s", settings.Cnf.Port)
		log.Println(fmt.Sprintf("Application starting at %v", address))
		if err := app.Listen(address); err != nil {
			log.Fatalf("Application could not started: %v", err)
		}
	}()

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	} else {
		log.Println("Server shutdown succeeded")
	}
}
