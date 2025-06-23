# Weather Query Service

## Overview

This project is a Go-based application that interacts with weather services to fetch temperature data for a given
location, returns average temperature and logs the results into a database.

## Features

- Fetches temperature data from multiple weather services asynchronously.
- Reduce third party service calls by batching requests made in last 5 seconds or 10 concurrent requests.
- Logs weather query results into the database.
- Includes unit tests for core functionality.
- Documentation provided via Swagger.

## Requirements

- Go 1.20 or later

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/ensarkovankaya/weather-app.git
   cd weather-app
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Setup required environment variables:
   ```plaintext
   PORT=8000
   WEATHER_API_URL=http://api.weatherapi.com/v1
   WEATHER_API_KEY=api_key_here
   WEATHER_STACK_API_URL=http://api.weatherstack.com
   WEATHER_STACK_API_KEY=api_key_here
   ```

## Usage

Run the application:

```bash
go run main.go
```

Run tests:

```bash
go test ./...
```

## Example Request

About the api endpoints see `api.yaml` swagger documentation.

```bash
curl http://localhost:8000/weather?q=London
```

## Example Response

```json
{
  "location": "London",
  "temperature": 15.5
}
```
