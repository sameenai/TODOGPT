package services

import (
	"testing"

	"github.com/todogpt/daily-briefing/internal/config"
)

func TestNewEmailService(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestEmailFetch(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{Enabled: true})

	emails, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(emails) == 0 {
		t.Error("expected mock emails")
	}

	for _, e := range emails {
		if e.ID == "" {
			t.Error("email should have an ID")
		}
		if e.From == "" {
			t.Error("email should have a from address")
		}
		if e.Subject == "" {
			t.Error("email should have a subject")
		}
		if e.Date.IsZero() {
			t.Error("email should have a date")
		}
	}
}

func TestEmailGetCachedEmpty(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	emails := svc.GetCached()
	if len(emails) == 0 {
		t.Error("expected mock emails when cache empty")
	}
}

func TestEmailUnreadCount(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	count := svc.UnreadCount()
	if count == 0 {
		t.Error("expected non-zero unread count from mock data")
	}
}

func TestEmailUnreadCountAfterFetch(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	svc.Fetch()
	count := svc.UnreadCount()
	if count == 0 {
		t.Error("expected non-zero unread count after fetch")
	}
}

func TestEmailMockHasUnreadAndStarred(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	emails := svc.mockEmails()

	hasUnread := false
	hasStarred := false
	for _, e := range emails {
		if e.IsUnread {
			hasUnread = true
		}
		if e.IsStarred {
			hasStarred = true
		}
	}

	if !hasUnread {
		t.Error("expected at least one unread email in mock data")
	}
	if !hasStarred {
		t.Error("expected at least one starred email in mock data")
	}
}

func TestEmailMockHasLabels(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	emails := svc.mockEmails()

	for _, e := range emails {
		if len(e.Labels) == 0 {
			t.Errorf("email %q should have labels", e.Subject)
		}
	}
}

func TestEmailGetCachedAfterFetch(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	svc.Fetch()
	cached := svc.GetCached()
	if len(cached) == 0 {
		t.Error("expected cached emails after fetch")
	}
	if cached[0].ID == "" {
		t.Error("cached email should have an ID")
	}
}
