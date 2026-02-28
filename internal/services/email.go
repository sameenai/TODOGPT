package services

import (
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

type EmailService struct {
	cfg   config.EmailConfig
	cache []models.EmailMessage
	mu    sync.RWMutex
}

func NewEmailService(cfg config.EmailConfig) *EmailService {
	return &EmailService{cfg: cfg}
}

func (s *EmailService) Fetch() ([]models.EmailMessage, error) {
	// When IMAP/Gmail credentials are configured, this would connect and fetch emails.
	msgs := s.mockEmails()
	s.mu.Lock()
	s.cache = msgs
	s.mu.Unlock()
	return msgs, nil
}

func (s *EmailService) GetCached() []models.EmailMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache != nil {
		return s.cache
	}
	return s.mockEmails()
}

func (s *EmailService) UnreadCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	emails := s.cache
	if emails == nil {
		emails = s.mockEmails()
	}
	for _, e := range emails {
		if e.IsUnread {
			count++
		}
	}
	return count
}

func (s *EmailService) mockEmails() []models.EmailMessage {
	now := time.Now()
	return []models.EmailMessage{
		{
			ID:       "email-1",
			From:     "boss@company.com",
			Subject:  "Q1 OKR Review - Action Required",
			Snippet:  "Please review and update your Q1 OKRs before Friday...",
			Date:     now.Add(-1 * time.Hour),
			IsUnread: true,
			Labels:   []string{"important", "inbox"},
		},
		{
			ID:       "email-2",
			From:     "github@notifications.github.com",
			Subject:  "[myorg/myrepo] PR #342 approved",
			Snippet:  "alex approved your pull request...",
			Date:     now.Add(-2 * time.Hour),
			IsUnread: true,
			Labels:   []string{"github", "inbox"},
		},
		{
			ID:       "email-3",
			From:     "noreply@linear.app",
			Subject:  "3 issues assigned to you",
			Snippet:  "You have been assigned: FE-234, FE-235, FE-236...",
			Date:     now.Add(-3 * time.Hour),
			IsUnread: true,
			Labels:   []string{"linear", "inbox"},
		},
		{
			ID:       "email-4",
			From:     "team-lead@company.com",
			Subject:  "Architecture Decision: Database Migration",
			Snippet:  "I'd like your input on the proposed migration from Postgres to...",
			Date:     now.Add(-5 * time.Hour),
			IsUnread: false,
			IsStarred: true,
			Labels:   []string{"important", "inbox"},
		},
		{
			ID:       "email-5",
			From:     "newsletter@techdigest.com",
			Subject:  "This Week in Tech: AI Updates",
			Snippet:  "Top stories: New LLM benchmarks, Rust 2.0 preview...",
			Date:     now.Add(-8 * time.Hour),
			IsUnread: true,
			Labels:   []string{"newsletter"},
		},
	}
}
