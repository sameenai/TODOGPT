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

func TestBuildPromptManyEvents(t *testing.T) {
	b := testBriefing()
	now := time.Now()
	// Add 4 events — triggers the count-only path (not names)
	b.Events = []models.CalendarEvent{
		{ID: "e1", Title: "Meeting A", StartTime: now.Add(1 * time.Hour)},
		{ID: "e2", Title: "Meeting B", StartTime: now.Add(2 * time.Hour)},
		{ID: "e3", Title: "Meeting C", StartTime: now.Add(3 * time.Hour)},
		{ID: "e4", Title: "Meeting D", StartTime: now.Add(4 * time.Hour)},
	}
	prompt := buildPrompt(b)
	if !strings.Contains(prompt, "4 events") {
		t.Errorf("expected '4 events' in prompt, got %q", prompt)
	}
}

func TestBuildPromptSlackUrgent(t *testing.T) {
	b := testBriefing()
	b.SlackMessages = []models.SlackMessage{
		{Channel: "#eng", User: "alice", Text: "urgent: server down", IsUrgent: true},
		{Channel: "#general", User: "bob", Text: "hey", IsUrgent: false},
	}
	prompt := buildPrompt(b)
	if !strings.Contains(prompt, "Slack") {
		t.Errorf("expected Slack section in prompt")
	}
	if !strings.Contains(prompt, "2 messages") {
		t.Errorf("expected 2 messages count in prompt")
	}
}

func TestBuildPromptGitHubUnread(t *testing.T) {
	b := testBriefing()
	b.GitHubNotifs = []models.GitHubNotification{
		{ID: "n1", Title: "PR merged", Repo: "org/repo", Unread: true},
	}
	prompt := buildPrompt(b)
	if !strings.Contains(prompt, "GitHub") {
		t.Errorf("expected GitHub section in prompt")
	}
}

func TestAIServiceEnvKeyFallback(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "env-key-xyz")
	svc := NewAIService(config.AIConfig{Enabled: true, APIKey: ""})
	if svc.apiKey() != "env-key-xyz" {
		t.Errorf("expected env key fallback, got %q", svc.apiKey())
	}
}

func TestAIServiceDefaultModelFallback(t *testing.T) {
	svc := NewAIService(config.AIConfig{Enabled: true, APIKey: "k", Model: ""})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{{"type": "text", "text": "ok"}},
		})
	}))
	defer ts.Close()
	orig := claudeAPIURL
	claudeAPIURL = ts.URL
	defer func() { claudeAPIURL = orig }()
	summary, _ := svc.Summarize(testBriefing())
	if summary != "ok" {
		t.Errorf("expected summary 'ok', got %q", summary)
	}
}

func TestAIServiceHandlesBadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()
	orig := claudeAPIURL
	claudeAPIURL = ts.URL
	defer func() { claudeAPIURL = orig }()
	svc := NewAIService(config.AIConfig{Enabled: true, APIKey: "k"})
	_, err := svc.Summarize(testBriefing())
	if err == nil {
		t.Error("expected error on bad JSON response")
	}
}

func TestAIServiceNetworkError(t *testing.T) {
	orig := claudeAPIURL
	claudeAPIURL = "http://127.0.0.1:1"
	defer func() { claudeAPIURL = orig }()
	svc := NewAIService(config.AIConfig{Enabled: true, APIKey: "k"})
	_, err := svc.Summarize(testBriefing())
	if err == nil {
		t.Error("expected network error")
	}
}

func TestNewTodoServiceFallsBackOnBadPath(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.DataDir = "/proc/nonexistent/path/that/cannot/exist"
	svc := newTodoService(cfg)
	if svc == nil {
		t.Fatal("expected non-nil service even on bad path")
	}
}

func TestFetchAllWithAISummaryDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.DataDir = t.TempDir()
	cfg.AI.Enabled = false
	hub := NewHub(cfg)
	briefing := hub.FetchAll()
	if briefing.Summary != "" {
		t.Errorf("expected empty summary when AI disabled, got %q", briefing.Summary)
	}
}

func TestFetchAllWithAISummarySuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"content": []map[string]string{{"type": "text", "text": "Good morning!"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	orig := claudeAPIURL
	claudeAPIURL = ts.URL
	defer func() { claudeAPIURL = orig }()

	cfg := config.DefaultConfig()
	cfg.Server.DataDir = t.TempDir()
	cfg.AI.Enabled = true
	cfg.AI.APIKey = "test-key"
	hub := NewHub(cfg)
	briefing := hub.FetchAll()
	if briefing.Summary != "Good morning!" {
		t.Errorf("expected 'Good morning!' summary, got %q", briefing.Summary)
	}
}

func TestFetchAllWithAISummaryError(t *testing.T) {
	orig := claudeAPIURL
	claudeAPIURL = "http://127.0.0.1:1" // unreachable port
	defer func() { claudeAPIURL = orig }()

	cfg := config.DefaultConfig()
	cfg.Server.DataDir = t.TempDir()
	cfg.AI.Enabled = true
	cfg.AI.APIKey = "test-key"
	hub := NewHub(cfg)
	briefing := hub.FetchAll()
	// AI error is silently logged; summary stays empty
	if briefing.Summary != "" {
		t.Errorf("expected empty summary on AI error, got %q", briefing.Summary)
	}
}
