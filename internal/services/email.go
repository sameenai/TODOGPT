package services

import (
	"fmt"
	"sync"
	"time"

	goImap "github.com/emersion/go-imap"
	imapClient "github.com/emersion/go-imap/client"
	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

// imapClientIface abstracts the go-imap client for testability.
type imapClientIface interface {
	Login(username, password string) error
	Select(name string, readOnly bool) (*goImap.MailboxStatus, error)
	Fetch(seqSet *goImap.SeqSet, items []goImap.FetchItem, ch chan *goImap.Message) error
	Logout() error
}

// imapDial is a package-level variable so tests can inject a fake client.
var imapDial = func(addr string) (imapClientIface, error) {
	return imapClient.DialTLS(addr, nil)
}

type EmailService struct {
	cfg   config.EmailConfig
	cache []models.EmailMessage
	mu    sync.RWMutex
}

func NewEmailService(cfg config.EmailConfig) *EmailService {
	return &EmailService{cfg: cfg}
}

// IsLive returns true when IMAP credentials are configured and enabled.
func (s *EmailService) IsLive() bool {
	return s.cfg.Enabled && s.cfg.Username != "" && s.cfg.Password != ""
}

func (s *EmailService) Fetch() ([]models.EmailMessage, error) {
	if !s.IsLive() {
		msgs := s.mockEmails()
		s.mu.Lock()
		s.cache = msgs
		s.mu.Unlock()
		return msgs, nil
	}

	msgs, err := s.fetchFromIMAP()
	if err != nil {
		s.mu.RLock()
		cached := s.cache
		s.mu.RUnlock()
		if cached != nil {
			return cached, nil
		}
		return s.mockEmails(), nil
	}

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

// ── IMAP ──────────────────────────────────────────────────────────────────────

func (s *EmailService) fetchFromIMAP() ([]models.EmailMessage, error) {
	addr := fmt.Sprintf("%s:%d", s.cfg.IMAPServer, s.cfg.IMAPPort)

	c, err := imapDial(addr)
	if err != nil {
		return nil, fmt.Errorf("imap dial: %w", err)
	}
	defer c.Logout() //nolint:errcheck

	if err := c.Login(s.cfg.Username, s.cfg.Password); err != nil {
		return nil, fmt.Errorf("imap login: %w", err)
	}

	mbox, err := c.Select("INBOX", true) // read-only
	if err != nil {
		return nil, fmt.Errorf("imap select INBOX: %w", err)
	}

	if mbox.Messages == 0 {
		return []models.EmailMessage{}, nil
	}

	// Fetch the 20 most recent messages
	from := uint32(1)
	if mbox.Messages > 20 {
		from = mbox.Messages - 19
	}
	seqSet := new(goImap.SeqSet)
	seqSet.AddRange(from, mbox.Messages)

	items := []goImap.FetchItem{goImap.FetchEnvelope, goImap.FetchFlags, goImap.FetchUid}
	messages := make(chan *goImap.Message, 20)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqSet, items, messages)
	}()

	var result []models.EmailMessage
	for msg := range messages {
		if msg.Envelope == nil {
			continue
		}
		env := msg.Envelope

		from := ""
		if len(env.From) > 0 {
			addr := env.From[0]
			if addr.PersonalName != "" {
				from = fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName)
			} else {
				from = fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
			}
		}

		isUnread := true
		isStarred := false
		for _, flag := range msg.Flags {
			if flag == goImap.SeenFlag {
				isUnread = false
			}
			if flag == goImap.FlaggedFlag {
				isStarred = true
			}
		}

		result = append(result, models.EmailMessage{
			ID:        fmt.Sprintf("imap-%d", msg.Uid),
			From:      from,
			Subject:   env.Subject,
			Snippet:   env.Subject, // envelope only; no body
			Date:      env.Date,
			IsUnread:  isUnread,
			IsStarred: isStarred,
			Labels:    []string{"inbox"},
		})
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("imap fetch: %w", err)
	}

	return result, nil
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
			ID:        "email-4",
			From:      "team-lead@company.com",
			Subject:   "Architecture Decision: Database Migration",
			Snippet:   "I'd like your input on the proposed migration from Postgres to...",
			Date:      now.Add(-5 * time.Hour),
			IsUnread:  false,
			IsStarred: true,
			Labels:    []string{"important", "inbox"},
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
