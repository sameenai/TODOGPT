package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

func testBriefing() *models.Briefing {
	now := time.Now()
	return &models.Briefing{
		Date:        now,
		GeneratedAt: now,
		Weather: &models.Weather{
			City:        "Test City",
			Temperature: 72,
			FeelsLike:   70,
			Humidity:    45,
			Description: "sunny",
			WindSpeed:   5,
		},
		Events: []models.CalendarEvent{
			{ID: "e1", Title: "Standup", StartTime: now.Add(time.Hour), EndTime: now.Add(2 * time.Hour)},
		},
		UnreadEmails: []models.EmailMessage{
			{ID: "m1", Subject: "Hello", IsUnread: true},
		},
		Todos: []models.TodoItem{
			{ID: "t1", Title: "Fix bug", Status: models.TodoPending, Priority: models.PriorityHigh},
		},
	}
}

func TestAIServiceDisabledReturnsEmpty(t *testing.T) {
	svc := NewAIService(config.AIConfig{Enabled: false})
	summary, err := svc.Summarize(testBriefing())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summary != "" {
		t.Errorf("expected empty summary when disabled, got %q", summary)
	}
}

func TestAIServiceNoKeyReturnsEmpty(t *testing.T) {
	svc := NewAIService(config.AIConfig{Enabled: true, APIKey: ""})
	t.Setenv("ANTHROPIC_API_KEY", "")
	summary, err := svc.Summarize(testBriefing())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summary != "" {
		t.Errorf("expected empty summary with no key, got %q", summary)
	}
}

func TestAIServiceCallsAPIAndReturnsSummary(t *testing.T) {
	wantText := "Good morning! Here is your daily briefing."

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key 'test-key', got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("expected anthropic-version header")
		}

		resp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": wantText},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	orig := claudeAPIURL
	claudeAPIURL = ts.URL
	defer func() { claudeAPIURL = orig }()

	svc := NewAIService(config.AIConfig{Enabled: true, APIKey: "test-key", Model: "claude-sonnet-4-6"})
	summary, err := svc.Summarize(testBriefing())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary != wantText {
		t.Errorf("expected %q, got %q", wantText, summary)
	}
}

func TestAIServiceHandlesAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer ts.Close()

	orig := claudeAPIURL
	claudeAPIURL = ts.URL
	defer func() { claudeAPIURL = orig }()

	svc := NewAIService(config.AIConfig{Enabled: true, APIKey: "bad-key"})
	_, err := svc.Summarize(testBriefing())
	if err == nil {
		t.Error("expected error on 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got: %v", err)
	}
}

func TestAIServiceHandlesEmptyContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"content": []map[string]string{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	orig := claudeAPIURL
	claudeAPIURL = ts.URL
	defer func() { claudeAPIURL = orig }()

	svc := NewAIService(config.AIConfig{Enabled: true, APIKey: "test-key"})
	summary, err := svc.Summarize(testBriefing())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary != "" {
		t.Errorf("expected empty summary for empty content, got %q", summary)
	}
}

func TestBuildPromptContainsKeyInfo(t *testing.T) {
	b := testBriefing()
	prompt := buildPrompt(b)

	if !strings.Contains(prompt, "Standup") {
		t.Error("expected calendar event 'Standup' in prompt")
	}
	if !strings.Contains(prompt, "72") {
		t.Error("expected temperature in prompt")
	}
	if !strings.Contains(prompt, "unread") {
		t.Error("expected email unread count in prompt")
	}
	if !strings.Contains(prompt, "Fix bug") {
		t.Error("expected todo title in prompt")
	}
}
