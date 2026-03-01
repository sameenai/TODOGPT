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

func TestWeatherCodeToDescriptionAllCodes(t *testing.T) {
	tests := []struct {
		code     int
		isDay    int
		wantDsc  string
		wantIcon string
	}{
		{0, 1, "clear sky", "01d"},
		{0, 0, "clear sky", "01n"},
		{1, 1, "mainly clear", "01d"},
		{2, 1, "partly cloudy", "02d"},
		{3, 1, "overcast", "04d"},
		{45, 1, "foggy", "50d"},
		{48, 0, "foggy", "50n"},
		{51, 1, "drizzle", "09d"},
		{53, 1, "drizzle", "09d"},
		{55, 1, "drizzle", "09d"},
		{61, 1, "rain", "10d"},
		{63, 1, "rain", "10d"},
		{65, 0, "rain", "10n"},
		{66, 1, "freezing rain", "13d"},
		{67, 1, "freezing rain", "13d"},
		{71, 1, "snow", "13d"},
		{73, 1, "snow", "13d"},
		{75, 1, "snow", "13d"},
		{77, 1, "snow grains", "13d"},
		{80, 1, "rain showers", "09d"},
		{81, 1, "rain showers", "09d"},
		{82, 1, "rain showers", "09d"},
		{85, 1, "snow showers", "13d"},
		{86, 0, "snow showers", "13n"},
		{95, 1, "thunderstorm", "11d"},
		{96, 1, "thunderstorm with hail", "11d"},
		{99, 1, "thunderstorm with hail", "11d"},
		{999, 1, "unknown", "01d"},
		{-1, 0, "unknown", "01n"},
	}

	for _, tc := range tests {
		desc, icon := weatherCodeToDescription(tc.code, tc.isDay)
		if desc != tc.wantDsc {
			t.Errorf("code %d isDay %d: expected desc %q, got %q", tc.code, tc.isDay, tc.wantDsc, desc)
		}
		if icon != tc.wantIcon {
			t.Errorf("code %d isDay %d: expected icon %q, got %q", tc.code, tc.isDay, tc.wantIcon, icon)
		}
	}
}

func TestApproximateFeelsLikeAllBranches(t *testing.T) {
	// Mild temperature — returns tempF as-is (else branch)
	fl := approximateFeelsLike(70, 5, 50, "imperial")
	if fl < 60 || fl > 80 {
		t.Errorf("expected feels-like near 70, got %.1f", fl)
	}

	// Cold + windy — wind chill (imperial)
	fl = approximateFeelsLike(30, 20, 30, "imperial")
	if fl >= 30 {
		t.Errorf("expected feels-like below 30 with wind chill, got %.1f", fl)
	}

	// Hot + humid — heat index (imperial)
	fl = approximateFeelsLike(95, 5, 70, "imperial")
	if fl <= 95 {
		t.Errorf("expected feels-like above 95 with high humidity, got %.1f", fl)
	}

	// Cold + windy — wind chill (metric, converts km/h to mph)
	fl = approximateFeelsLike(-5, 30, 30, "metric")
	if fl >= -5 {
		t.Errorf("expected feels-like below -5C with wind chill (metric), got %.1f", fl)
	}

	// Hot + humid — heat index (metric, converts C to F and back)
	fl = approximateFeelsLike(35, 5, 70, "metric")
	if fl <= 35 {
		t.Errorf("expected feels-like above 35C with high humidity (metric), got %.1f", fl)
	}

	// Mild metric — should return near input
	fl = approximateFeelsLike(20, 2, 50, "metric")
	if fl < 15 || fl > 25 {
		t.Errorf("expected feels-like near 20C, got %.1f", fl)
	}
}

func TestWeatherFetchWithCoordinates(t *testing.T) {
	svc := NewWeatherService(config.WeatherConfig{
		City:  "Berlin",
		Units: "metric",
		Lat:   52.52,
		Lon:   13.405,
	})

	w, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected weather data")
	}
}

func TestWeatherFetchEmptyCity(t *testing.T) {
	svc := NewWeatherService(config.WeatherConfig{
		City:  "",
		Units: "imperial",
	})

	w, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected weather data")
	}
}

func TestWeatherFetchMetricUnits(t *testing.T) {
	svc := NewWeatherService(config.WeatherConfig{
		City:  "London",
		Units: "metric",
	})

	w, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected weather data")
	}
}
