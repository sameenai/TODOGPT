package services

import (
	"testing"

	"github.com/todogpt/daily-briefing/internal/config"
)

func TestNewWeatherService(t *testing.T) {
	cfg := config.WeatherConfig{City: "Berlin", Units: "metric"}
	svc := NewWeatherService(cfg)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestWeatherFetchReturnsData(t *testing.T) {
	svc := NewWeatherService(config.WeatherConfig{
		City:  "Paris",
		Units: "metric",
	})

	w, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected weather data, got nil")
	}
	if w.Temperature == 0 && w.Description == "" {
		t.Error("expected non-zero temperature or description")
	}
}

func TestWeatherGetCachedEmpty(t *testing.T) {
	svc := NewWeatherService(config.WeatherConfig{
		City:  "London",
		Units: "metric",
	})

	w := svc.GetCached()
	if w == nil {
		t.Fatal("expected mock weather when cache empty")
	}
	if w.City != "London" {
		t.Errorf("expected city London, got %s", w.City)
	}
}

func TestWeatherGetCachedAfterFetch(t *testing.T) {
	svc := NewWeatherService(config.WeatherConfig{
		City:  "Tokyo",
		Units: "imperial",
	})

	svc.Fetch()
	cached := svc.GetCached()
	if cached == nil {
		t.Fatal("expected cached weather")
	}
}

func TestWeatherMockDataConsistency(t *testing.T) {
	svc := NewWeatherService(config.WeatherConfig{
		City:    "SF",
		Units:   "imperial",
		Enabled: true,
	})

	w := svc.mockWeather()
	if w.Units != "imperial" {
		t.Errorf("mock weather units should match config, got %s", w.Units)
	}
	if w.City != "SF" {
		t.Errorf("mock weather city should match config, got %s", w.City)
	}
	if w.UpdatedAt.IsZero() {
		t.Error("mock weather should have non-zero UpdatedAt")
	}
}

func TestWeatherCodeToDescription(t *testing.T) {
	tests := []struct {
		code    int
		isDay   int
		wantDsc string
	}{
		{0, 1, "clear sky"},
		{2, 1, "partly cloudy"},
		{3, 0, "overcast"},
		{61, 1, "rain"},
		{95, 0, "thunderstorm"},
	}

	for _, tc := range tests {
		desc, _ := weatherCodeToDescription(tc.code, tc.isDay)
		if desc != tc.wantDsc {
			t.Errorf("code %d: expected %q, got %q", tc.code, tc.wantDsc, desc)
		}
	}
}

func TestApproximateFeelsLike(t *testing.T) {
	// Mild temperature should return approximately the same
	fl := approximateFeelsLike(70, 5, 50, "imperial")
	if fl < 60 || fl > 80 {
		t.Errorf("expected feels-like near 70, got %.1f", fl)
	}

	// Cold + windy should feel colder
	fl = approximateFeelsLike(30, 20, 30, "imperial")
	if fl >= 30 {
		t.Errorf("expected feels-like below 30 with wind chill, got %.1f", fl)
	}
}
