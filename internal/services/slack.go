package services

import (
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

type SlackService struct {
	cfg   config.SlackConfig
	cache []models.SlackMessage
	mu    sync.RWMutex
}

func NewSlackService(cfg config.SlackConfig) *SlackService {
	return &SlackService{cfg: cfg}
}

func (s *SlackService) Fetch() ([]models.SlackMessage, error) {
	// When Slack tokens are configured, this would connect to the Slack API
	// using the Bot Token for reading messages and App Token for Socket Mode.
	msgs := s.mockMessages()
	s.mu.Lock()
	s.cache = msgs
	s.mu.Unlock()
	return msgs, nil
}

func (s *SlackService) GetCached() []models.SlackMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache != nil {
		return s.cache
	}
	return s.mockMessages()
}

func (s *SlackService) mockMessages() []models.SlackMessage {
	now := time.Now()
	return []models.SlackMessage{
		{
			Channel:   "#engineering",
			User:      "sarah",
			Text:      "Deployed v2.3.1 to staging. Can someone verify the auth flow?",
			Timestamp: now.Add(-30 * time.Minute),
			IsUrgent:  true,
		},
		{
			Channel:   "#general",
			User:      "mike",
			Text:      "Team lunch moved to 12:30 today",
			Timestamp: now.Add(-1 * time.Hour),
		},
		{
			Channel:   "DM",
			User:      "alex",
			Text:      "Hey, can you review my PR when you get a chance?",
			Timestamp: now.Add(-2 * time.Hour),
			IsDM:      true,
		},
		{
			Channel:   "#incidents",
			User:      "pagerduty-bot",
			Text:      "RESOLVED: API latency spike on us-east-1 has been mitigated",
			Timestamp: now.Add(-3 * time.Hour),
			IsUrgent:  true,
		},
		{
			Channel:   "#random",
			User:      "jordan",
			Text:      "Anyone up for a coffee run at 3pm?",
			Timestamp: now.Add(-4 * time.Hour),
		},
	}
}
