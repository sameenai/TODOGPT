package services

import (
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

type CalendarService struct {
	cfg   config.GoogleConfig
	cache []models.CalendarEvent
	mu    sync.RWMutex
}

func NewCalendarService(cfg config.GoogleConfig) *CalendarService {
	return &CalendarService{cfg: cfg}
}

func (s *CalendarService) Fetch() ([]models.CalendarEvent, error) {
	// When Google credentials are configured, this would use the Google Calendar API.
	// For now, return demo events to show the dashboard layout.
	events := s.mockEvents()
	s.mu.Lock()
	s.cache = events
	s.mu.Unlock()
	return events, nil
}

func (s *CalendarService) GetCached() []models.CalendarEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache != nil {
		return s.cache
	}
	return s.mockEvents()
}

func (s *CalendarService) mockEvents() []models.CalendarEvent {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	return []models.CalendarEvent{
		{
			ID:        "evt-1",
			Title:     "Morning Standup",
			StartTime: today.Add(9 * time.Hour),
			EndTime:   today.Add(9*time.Hour + 15*time.Minute),
			Location:  "Zoom",
			MeetingURL: "https://zoom.us/j/example",
			Attendees: []string{"team@company.com"},
			Source:    "google_calendar",
		},
		{
			ID:          "evt-2",
			Title:       "Sprint Planning",
			Description: "Q1 sprint planning session",
			StartTime:   today.Add(10 * time.Hour),
			EndTime:     today.Add(11 * time.Hour),
			Location:    "Conference Room A",
			Attendees:   []string{"engineering@company.com"},
			Source:      "google_calendar",
		},
		{
			ID:        "evt-3",
			Title:     "Lunch with Alex",
			StartTime: today.Add(12 * time.Hour),
			EndTime:   today.Add(13 * time.Hour),
			Location:  "Downtown Cafe",
			Source:    "google_calendar",
		},
		{
			ID:          "evt-4",
			Title:       "Code Review Session",
			Description: "Review PR #342 and #358",
			StartTime:   today.Add(14 * time.Hour),
			EndTime:     today.Add(15 * time.Hour),
			MeetingURL:  "https://meet.google.com/abc-defg-hij",
			Source:      "google_calendar",
		},
		{
			ID:        "evt-5",
			Title:     "1:1 with Manager",
			StartTime: today.Add(16 * time.Hour),
			EndTime:   today.Add(16*time.Hour + 30*time.Minute),
			MeetingURL: "https://zoom.us/j/example2",
			Source:    "google_calendar",
		},
	}
}
