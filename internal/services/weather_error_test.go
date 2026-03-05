package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/todogpt/daily-briefing/internal/config"
)

// restoreWeatherURLs returns a cleanup func that resets the URL vars.
func restoreWeatherURLs() func() {
	origMeteo := openMeteoBaseURL
	origGeo := geocodingBaseURL
	return func() {
		openMeteoBaseURL = origMeteo
		geocodingBaseURL = origGeo
	}
}

func TestWeatherFetchNonOKStatus(t *testing.T) {
	defer restoreWeatherURLs()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()
	openMeteoBaseURL = ts.URL

	svc := NewWeatherService(config.WeatherConfig{
		City: "NYC", Units: "imperial", Lat: 40.7, Lon: -74.0,
	})
	w, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Falls back to mock on non-200
	if w == nil {
		t.Fatal("expected mock weather on non-200 status")
	}
}

func TestWeatherFetchInvalidJSON(t *testing.T) {
	defer restoreWeatherURLs()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json{{"))
	}))
	defer ts.Close()
	openMeteoBaseURL = ts.URL

	svc := NewWeatherService(config.WeatherConfig{
		City: "NYC", Units: "imperial", Lat: 40.7, Lon: -74.0,
	})
	w, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected mock weather on JSON decode failure")
	}
}

func TestWeatherFetchHumidityIndexOutOfRange(t *testing.T) {
	defer restoreWeatherURLs()()

	payload := openMeteoResponse{}
	payload.CurrentWeather.Temperature = 72
	payload.CurrentWeather.WindSpeed = 5
	payload.CurrentWeather.WeatherCode = 0
	payload.CurrentWeather.IsDay = 1
	// Provide exactly 1 humidity value; current hour may exceed index
	payload.Hourly.RelativeHumidity = []int{50}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer ts.Close()
	openMeteoBaseURL = ts.URL

	svc := NewWeatherService(config.WeatherConfig{
		City: "NYC", Units: "imperial", Lat: 40.7, Lon: -74.0,
	})
	w, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected weather data")
	}
}

func TestGeocodeNonOKStatus(t *testing.T) {
	defer restoreWeatherURLs()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer ts.Close()
	geocodingBaseURL = ts.URL

	svc := NewWeatherService(config.WeatherConfig{City: "UnknownCity", Units: "imperial"})
	w, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error (should fall back to mock): %v", err)
	}
	if w == nil {
		t.Fatal("expected mock weather when geocode fails")
	}
}

func TestGeocodeNoResults(t *testing.T) {
	defer restoreWeatherURLs()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer ts.Close()
	geocodingBaseURL = ts.URL

	svc := NewWeatherService(config.WeatherConfig{City: "GhostCity", Units: "imperial"})
	w, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected mock weather when geocode returns no results")
	}
}

func TestGeocodeInvalidJSON(t *testing.T) {
	defer restoreWeatherURLs()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("bad json{{"))
	}))
	defer ts.Close()
	geocodingBaseURL = ts.URL

	svc := NewWeatherService(config.WeatherConfig{City: "AnyCity", Units: "imperial"})
	w, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected mock weather on geocode JSON error")
	}
}
