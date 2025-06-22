package settings

import (
	"github.com/Netflix/go-env"
	"log"
)

var Cnf Config

type Config struct {
	Port string `env:"PORT,default=8000"`

	WeatherAPIURL string `env:"WEATHER_API_URL,default=http://api.weatherapi.com/v1"`
	WeatherAPIKey string `env:"WEATHER_API_KEY"`

	WeatherStackAPIURL string `env:"WEATHER_STACK_API_URL,default=http://api.weatherstack.com/v1"`
	WeatherStackAPIKey string `env:"WEATHER_STACK_API_KEY"`
}

func init() {
	_, err := env.UnmarshalFromEnviron(&Cnf)
	if err != nil {
		log.Fatal(err)
	}
}
