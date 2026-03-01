package services

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

type WeatherService struct {
	cfg   config.WeatherConfig
	cache *models.Weather
	mu    sync.RWMutex
}

func NewWeatherService(cfg config.WeatherConfig) *WeatherService {
	return &WeatherService{cfg: cfg}
}

// geocodeResponse is used to resolve a city name to coordinates via Open-Meteo.
type geocodeResponse struct {
	Results []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Name      string  `json:"name"`
		Country   string  `json:"country"`
	} `json:"results"`
}

// openMeteoResponse is the response from the Open-Meteo current weather API.
type openMeteoResponse struct {
	CurrentWeather struct {
		Temperature float64 `json:"temperature"`
		WindSpeed   float64 `json:"windspeed"`
		WeatherCode int     `json:"weathercode"`
		IsDay       int     `json:"is_day"`
	} `json:"current_weather"`
	Hourly struct {
		RelativeHumidity []int `json:"relative_humidity_2m"`
	} `json:"hourly"`
}

func (s *WeatherService) Fetch() (*models.Weather, error) {
	lat, lon := s.cfg.Lat, s.cfg.Lon
	city := s.cfg.City

	// If no coordinates, geocode the city name
	if lat == 0 && lon == 0 {
		if city == "" {
			city = "New York"
		}
		var err error
		lat, lon, city, err = geocodeCity(city)
		if err != nil {
			return s.mockWeather(), nil
		}
	}

	units := "fahrenheit"
	windUnit := "mph"
	if s.cfg.Units == "metric" {
		units = "celsius"
		windUnit = "kmh"
	}

	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current_weather=true&hourly=relative_humidity_2m&temperature_unit=%s&wind_speed_unit=%s&timezone=auto",
		lat, lon, units, windUnit,
	)

	resp, err := http.Get(url)
	if err != nil {
		return s.mockWeather(), nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return s.mockWeather(), nil
	}

	var omResp openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&omResp); err != nil {
		return s.mockWeather(), nil
	}

	humidity := 0
	if len(omResp.Hourly.RelativeHumidity) > 0 {
		hour := time.Now().Hour()
		if hour < len(omResp.Hourly.RelativeHumidity) {
			humidity = omResp.Hourly.RelativeHumidity[hour]
		} else {
			humidity = omResp.Hourly.RelativeHumidity[0]
		}
	}

	desc, icon := weatherCodeToDescription(omResp.CurrentWeather.WeatherCode, omResp.CurrentWeather.IsDay)
	feelsLike := approximateFeelsLike(omResp.CurrentWeather.Temperature, omResp.CurrentWeather.WindSpeed, humidity, s.cfg.Units)

	w := &models.Weather{
		City:        city,
		Temperature: omResp.CurrentWeather.Temperature,
		FeelsLike:   feelsLike,
		Humidity:    humidity,
		Description: desc,
		Icon:        icon,
		WindSpeed:   omResp.CurrentWeather.WindSpeed,
		Units:       s.cfg.Units,
		UpdatedAt:   time.Now(),
	}

	s.mu.Lock()
	s.cache = w
	s.mu.Unlock()

	return w, nil
}

func (s *WeatherService) GetCached() *models.Weather {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache != nil {
		return s.cache
	}
	return s.mockWeather()
}

func (s *WeatherService) mockWeather() *models.Weather {
	return &models.Weather{
		City:        s.cfg.City,
		Temperature: 72,
		FeelsLike:   70,
		Humidity:    45,
		Description: "partly cloudy",
		Icon:        "02d",
		WindSpeed:   8.5,
		Units:       s.cfg.Units,
		UpdatedAt:   time.Now(),
	}
}

func geocodeCity(city string) (lat, lon float64, name string, err error) {
	url := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=en&format=json", city)
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, city, fmt.Errorf("geocode error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var geoResp geocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&geoResp); err != nil {
		return 0, 0, city, fmt.Errorf("geocode decode error: %w", err)
	}

	if len(geoResp.Results) == 0 {
		return 0, 0, city, fmt.Errorf("city not found: %s", city)
	}

	r := geoResp.Results[0]
	return r.Latitude, r.Longitude, r.Name, nil
}

// weatherCodeToDescription maps WMO weather codes to human descriptions and icon codes.
func weatherCodeToDescription(code int, isDay int) (string, string) {
	dayNight := "d"
	if isDay == 0 {
		dayNight = "n"
	}

	switch code {
	case 0:
		return "clear sky", "01" + dayNight
	case 1:
		return "mainly clear", "01" + dayNight
	case 2:
		return "partly cloudy", "02" + dayNight
	case 3:
		return "overcast", "04" + dayNight
	case 45, 48:
		return "foggy", "50" + dayNight
	case 51, 53, 55:
		return "drizzle", "09" + dayNight
	case 61, 63, 65:
		return "rain", "10" + dayNight
	case 66, 67:
		return "freezing rain", "13" + dayNight
	case 71, 73, 75:
		return "snow", "13" + dayNight
	case 77:
		return "snow grains", "13" + dayNight
	case 80, 81, 82:
		return "rain showers", "09" + dayNight
	case 85, 86:
		return "snow showers", "13" + dayNight
	case 95:
		return "thunderstorm", "11" + dayNight
	case 96, 99:
		return "thunderstorm with hail", "11" + dayNight
	default:
		return "unknown", "01" + dayNight
	}
}

// approximateFeelsLike provides a simple feels-like temperature approximation.
func approximateFeelsLike(temp, windSpeed float64, humidity int, units string) float64 {
	tempF := temp
	if units == "metric" {
		tempF = temp*9/5 + 32
	}

	var feelsLikeF float64
	if tempF <= 50 && windSpeed > 3 {
		windMph := windSpeed
		if units == "metric" {
			windMph = windSpeed * 0.621371
		}
		feelsLikeF = 35.74 + 0.6215*tempF - 35.75*math.Pow(windMph, 0.16) + 0.4275*tempF*math.Pow(windMph, 0.16)
	} else if tempF >= 80 && humidity > 40 {
		h := float64(humidity)
		feelsLikeF = -42.379 + 2.04901523*tempF + 10.14333127*h -
			0.22475541*tempF*h - 6.83783e-3*tempF*tempF -
			5.481717e-2*h*h + 1.22874e-3*tempF*tempF*h +
			8.5282e-4*tempF*h*h - 1.99e-6*tempF*tempF*h*h
	} else {
		feelsLikeF = tempF
	}

	if units == "metric" {
		return (feelsLikeF - 32) * 5 / 9
	}
	return feelsLikeF
}
