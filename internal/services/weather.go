package services

import (
	"encoding/json"
	"fmt"
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

type openWeatherResponse struct {
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Humidity  int     `json:"humidity"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Wind struct {
		Speed float64 `json:"speed"`
	} `json:"wind"`
	Name string `json:"name"`
}

func (s *WeatherService) Fetch() (*models.Weather, error) {
	if s.cfg.APIKey == "" {
		return s.mockWeather(), nil
	}

	query := s.cfg.City
	if s.cfg.Country != "" {
		query = s.cfg.City + "," + s.cfg.Country
	}

	url := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=%s",
		query, s.cfg.APIKey, s.cfg.Units,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("weather API error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var owResp openWeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&owResp); err != nil {
		return nil, fmt.Errorf("weather decode error: %w", err)
	}

	desc := ""
	icon := ""
	if len(owResp.Weather) > 0 {
		desc = owResp.Weather[0].Description
		icon = owResp.Weather[0].Icon
	}

	w := &models.Weather{
		City:        owResp.Name,
		Temperature: owResp.Main.Temp,
		FeelsLike:   owResp.Main.FeelsLike,
		Humidity:    owResp.Main.Humidity,
		Description: desc,
		Icon:        icon,
		WindSpeed:   owResp.Wind.Speed,
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
