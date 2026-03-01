package services

import (
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
)

func TestNewCalendarService(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestCalendarFetch(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{CalendarEnabled: true})

	events, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) == 0 {
		t.Error("expected mock events")
	}

	for _, evt := range events {
		if evt.ID == "" {
			t.Error("event should have an ID")
		}
		if evt.Title == "" {
			t.Error("event should have a title")
		}
		if evt.StartTime.IsZero() {
			t.Error("event should have a start time")
		}
		if evt.EndTime.IsZero() {
			t.Error("event should have an end time")
		}
		if evt.Source != "google_calendar" {
			t.Errorf("expected source google_calendar, got %s", evt.Source)
		}
	}
}

func TestCalendarGetCachedEmpty(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	events := svc.GetCached()
	if len(events) == 0 {
		t.Error("expected mock events when cache empty")
	}
}

func TestCalendarGetCachedAfterFetch(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	svc.Fetch()
	cached := svc.GetCached()
	if len(cached) == 0 {
		t.Error("expected cached events")
	}
}

func TestCalendarMockEventsAreToday(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	events := svc.mockEvents()

	today := time.Now()
	for _, evt := range events {
		if evt.StartTime.Year() != today.Year() ||
			evt.StartTime.Month() != today.Month() ||
			evt.StartTime.Day() != today.Day() {
			t.Errorf("event %q start time should be today, got %s", evt.Title, evt.StartTime)
		}
	}
}

func TestCalendarMockEventsCount(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	events := svc.mockEvents()
	if len(events) != 5 {
		t.Errorf("expected 5 mock events, got %d", len(events))
	}
}

func TestCalendarEventEndAfterStart(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	events := svc.mockEvents()

	for _, evt := range events {
		if !evt.EndTime.After(evt.StartTime) {
			t.Errorf("event %q: end time should be after start time", evt.Title)
		}
	}
}
