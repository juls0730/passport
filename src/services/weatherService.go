package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type WeatherProvider string

const (
	OpenWeatherMap WeatherProvider = "openweathermap"
)

type DepricatedWeatherConfig struct {
	OpenWeather struct {
		Provider WeatherProvider `env:"OPENWEATHER_PROVIDER" envDefault:"openweathermap"`
		APIKey   string          `env:"OPENWEATHER_API_KEY"`
		Units    string          `env:"OPENWEATHER_TEMP_UNITS" envDefault:"metric"`
		Lat      float64         `env:"OPENWEATHER_LAT"`
		Lon      float64         `env:"OPENWEATHER_LON"`
	}
	UpdateInterval int `env:"OPENWEATHER_UPDATE_INTERVAL" envDefault:"15"`
}

type WeatherConfig struct {
	Provider       WeatherProvider `env:"WEATHER_PROVIDER" envDefault:"openweathermap"`
	APIKey         string          `env:"WEATHER_API_KEY"`
	Units          string          `env:"WEATHER_TEMP_UNITS" envDefault:"metric"`
	Lat            float64         `env:"WEATHER_LAT"`
	Lon            float64         `env:"WEATHER_LON"`
	UpdateInterval int             `env:"WEATHER_UPDATE_INTERVAL" envDefault:"15"`
}

type OpenWeatherResponse struct {
	Weather []struct {
		Name   string `json:"main"`
		IconId string `json:"icon"`
	} `json:"weather"`
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Code    int    `json:"cod"`
	Message string `json:"message"`
}

type WeatherData struct {
	Temperature float64
	WeatherText string
	Icon        string
}

type WeatherManager struct {
	data       *WeatherData
	lastUpdate time.Time
	mutex      sync.RWMutex
	updateChan chan struct{}
	config     *WeatherConfig
}

func NewWeatherManager(config *WeatherConfig) *WeatherManager {
	if config.Provider != OpenWeatherMap {
		log.Fatalln("Only OpenWeatherMap is supported!")
		return nil
	}

	if config.APIKey == "" {
		log.Fatalln("An API Key required for OpenWeather!")
		return nil
	}

	updateInterval := config.UpdateInterval
	if updateInterval < 1 {
		updateInterval = 15
	}

	units := config.Units
	if units == "" {
		units = "metric"
	}

	cache := &WeatherManager{
		data:       &WeatherData{},
		updateChan: make(chan struct{}),
		config:     config,
	}

	go cache.weatherWorker()

	cache.updateChan <- struct{}{}

	return cache
}

func (c *WeatherManager) GetWeather() WeatherData {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return *c.data
}

func (c *WeatherManager) weatherWorker() {
	ticker := time.NewTicker(time.Duration(c.config.UpdateInterval) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.updateChan:
			c.updateWeather()
		case <-ticker.C:
			c.updateWeather()
		}
	}
}

func (c *WeatherManager) updateWeather() {
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s&units=%s",
		c.config.Lat, c.config.Lon, c.config.APIKey, c.config.Units)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching weather: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	var weatherResp OpenWeatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		fmt.Printf("Error parsing weather data: %v\n", err)
		return
	}

	// if the request failed
	if weatherResp.Code != 200 {
		// if there is no pre-existing data in the cache
		if c.data.WeatherText == "" {
			log.Fatalf("Fetching the weather data failed!\n%s\n", weatherResp.Message)
		} else {
			return
		}
	}

	c.mutex.Lock()
	c.data.Temperature = weatherResp.Main.Temp
	c.data.WeatherText = weatherResp.Weather[0].Name
	c.data.Icon = weatherResp.Weather[0].IconId
	c.lastUpdate = time.Now()
	c.mutex.Unlock()
}
