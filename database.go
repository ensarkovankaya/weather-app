package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func initializeDatabase() (*sql.DB, error) {
	var err error
	var db *sql.DB
	if db, err = sql.Open("sqlite3", "weather.sqlite"); err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	// Create weather_queries table if it doesn't exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS weather_queries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		location TEXT,
		service_1_temperature REAL,
		service_2_temperature REAL,
		request_count INTEGER,
		created_at TEXT
	)`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}
	return db, nil
}
