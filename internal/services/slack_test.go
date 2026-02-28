package services

import (
	"testing"

	"github.com/todogpt/daily-briefing/internal/config"
)

func TestNewSlackService(t *testing.T) {
	svc := NewSlackService(config.SlackConfig{})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestSlackFetch(t *testing.T) {
	svc := NewSlackService(config.SlackConfig{Enabled: true})

	msgs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Error("expected mock messages")
	}

	for _, m := range msgs {
		if m.Channel == "" {
			t.Error("message should have a channel")
		}
		if m.User == "" {
			t.Error("message should have a user")
		}
		if m.Text == "" {
			t.Error("message should have text")
		}
		if m.Timestamp.IsZero() {
			t.Error("message should have a timestamp")
		}
	}
}

func TestSlackGetCachedEmpty(t *testing.T) {
	svc := NewSlackService(config.SlackConfig{})
	msgs := svc.GetCached()
	if len(msgs) == 0 {
		t.Error("expected mock messages when cache empty")
	}
}

func TestSlackGetCachedAfterFetch(t *testing.T) {
	svc := NewSlackService(config.SlackConfig{})
	svc.Fetch()
	cached := svc.GetCached()
	if len(cached) == 0 {
		t.Error("expected cached messages")
	}
}

func TestSlackMockHasUrgentAndDM(t *testing.T) {
	svc := NewSlackService(config.SlackConfig{})
	msgs := svc.mockMessages()

	hasUrgent := false
	hasDM := false
	for _, m := range msgs {
		if m.IsUrgent {
			hasUrgent = true
		}
		if m.IsDM {
			hasDM = true
		}
	}

	if !hasUrgent {
		t.Error("expected at least one urgent message in mock data")
	}
	if !hasDM {
		t.Error("expected at least one DM in mock data")
	}
}

func TestSlackMockCount(t *testing.T) {
	svc := NewSlackService(config.SlackConfig{})
	msgs := svc.mockMessages()
	if len(msgs) != 5 {
		t.Errorf("expected 5 mock messages, got %d", len(msgs))
	}
}
