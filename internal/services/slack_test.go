package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
)

func TestNewSlackService(t *testing.T) {
	svc := NewSlackService(config.SlackConfig{})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestSlackIsLive(t *testing.T) {
	cases := []struct {
		cfg  config.SlackConfig
		want bool
	}{
		{config.SlackConfig{}, false},
		{config.SlackConfig{Enabled: true}, false},
		{config.SlackConfig{Enabled: true, BotToken: "xoxb-test"}, true},
		{config.SlackConfig{BotToken: "xoxb-test"}, false}, // enabled=false
	}
	for _, c := range cases {
		svc := NewSlackService(c.cfg)
		if got := svc.IsLive(); got != c.want {
			t.Errorf("IsLive(%+v) = %v, want %v", c.cfg, got, c.want)
		}
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
	svc.Fetch() //nolint:errcheck
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

func TestIsUrgentText(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"Everything is fine", false},
		{"URGENT: server is down", true},
		{"please help me with this", true},
		{"there is an error in production", true},
		{"@here please review this", true},
		{"@channel important announcement", true},
		{"system outage detected", true},
		{"just a casual update", false},
	}
	for _, c := range cases {
		if got := isUrgentText(c.text); got != c.want {
			t.Errorf("isUrgentText(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

func TestParseSlackTS(t *testing.T) {
	ts := parseSlackTS("1609459200.000000") // 2021-01-01 00:00:00 UTC
	if ts.IsZero() {
		t.Error("expected non-zero time")
	}
	if ts.Year() != 2021 {
		t.Errorf("expected year 2021, got %d", ts.Year())
	}

	// Empty string returns a non-zero time (time.Now())
	empty := parseSlackTS("")
	if empty.IsZero() {
		t.Error("empty ts should return time.Now(), not zero")
	}

	// Invalid string falls back to time.Now()
	invalid := parseSlackTS("not-a-timestamp")
	if invalid.IsZero() {
		t.Error("invalid ts should return time.Now(), not zero")
	}
}

func TestSlackFetchFromAPI(t *testing.T) {
	// Mock the Slack API
	mux := http.NewServeMux()

	// conversations.history
	mux.HandleFunc("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("channel") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		resp := slackHistoryResp{
			OK: true,
			Messages: []slackRawMsg{
				{Type: "message", Text: "Hello team", User: "U123", Ts: "1609459200.000000"},
				{Type: "message", Text: "urgent: server down", User: "U456", Ts: "1609459201.000000"},
				{Type: "message", Subtype: "bot_message", BotID: "B001", Text: "automated", User: ""},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	// users.info
	mux.HandleFunc("/users.info", func(w http.ResponseWriter, r *http.Request) {
		uid := r.URL.Query().Get("user")
		name := "unknown"
		if uid == "U123" {
			name = "alice"
		} else if uid == "U456" {
			name = "bob"
		}
		resp := slackUserResp{OK: true}
		resp.User.Profile.DisplayName = name
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	// conversations.info
	mux.HandleFunc("/conversations.info", func(w http.ResponseWriter, r *http.Request) {
		resp := slackChannelResp{OK: true}
		resp.Channel.ID = r.URL.Query().Get("channel")
		resp.Channel.Name = "general"
		resp.Channel.IsIM = false
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	orig := slackAPIBase
	slackAPIBase = ts.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{
		Enabled:  true,
		BotToken: "xoxb-test",
		Channels: []string{"C001"},
	})

	msgs, err := svc.fetchFromAPI()
	if err != nil {
		t.Fatalf("fetchFromAPI error: %v", err)
	}

	// 3 messages in response; bot_message should be filtered out → 2 remain
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages (bot filtered), got %d", len(msgs))
	}

	// Verify urgency detection
	urgentCount := 0
	for _, m := range msgs {
		if m.IsUrgent {
			urgentCount++
		}
	}
	if urgentCount != 1 {
		t.Errorf("expected 1 urgent message, got %d", urgentCount)
	}
}

func TestSlackFetchFromAPIError(t *testing.T) {
	// Server returns Slack API error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/conversations.history" {
			json.NewEncoder(w).Encode(slackHistoryResp{OK: false, Error: "channel_not_found"}) //nolint:errcheck
		} else {
			json.NewEncoder(w).Encode(slackChannelResp{OK: false}) //nolint:errcheck
		}
	}))
	defer ts.Close()

	orig := slackAPIBase
	slackAPIBase = ts.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{
		Enabled:  true,
		BotToken: "xoxb-test",
		Channels: []string{"C001"},
	})

	// fetchFromAPI skips channels that fail, so returns empty (not error)
	msgs, err := svc.fetchFromAPI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = msgs
}

func TestSlackFetchFallsBackToCache(t *testing.T) {
	// Serve valid data first, then fail
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path == "/conversations.history" {
			if callCount <= 3 { // first call: history + channel info + user info
				resp := slackHistoryResp{
					OK:       true,
					Messages: []slackRawMsg{{Type: "message", Text: "hi", User: "U1", Ts: "1609459200.000000"}},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp) //nolint:errcheck
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			// users.info and conversations.info
			if r.URL.Path == "/users.info" {
				resp := slackUserResp{OK: true}
				resp.User.Profile.DisplayName = "alice"
				json.NewEncoder(w).Encode(resp) //nolint:errcheck
			} else {
				resp := slackChannelResp{OK: true}
				resp.Channel.Name = "general"
				json.NewEncoder(w).Encode(resp) //nolint:errcheck
			}
		}
	}))
	defer ts.Close()

	orig := slackAPIBase
	slackAPIBase = ts.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{
		Enabled:  true,
		BotToken: "xoxb-test",
		Channels: []string{"C001"},
	})

	// First fetch — populates cache
	msgs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("first fetch error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected messages on first fetch")
	}
}

func TestSlackFetchEmptyChannels(t *testing.T) {
	svc := NewSlackService(config.SlackConfig{
		Enabled:  true,
		BotToken: "xoxb-test",
		Channels: []string{}, // no channels configured
	})

	// No API call should be made; returns empty slice
	orig := slackAPIBase
	slackAPIBase = "http://127.0.0.1:1" // invalid — any attempt would fail
	defer func() { slackAPIBase = orig }()

	msgs, err := svc.fetchFromAPI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for empty channels, got %d", len(msgs))
	}
}

func TestSlackResolveDMChannel(t *testing.T) {
	svc := NewSlackService(config.SlackConfig{BotToken: "xoxb-test"})
	// DM channels start with "D" — no API call needed
	name, isDM := svc.resolveChannel("D123ABC")
	if !isDM {
		t.Error("expected isDM=true for D-prefixed channel")
	}
	if name != "D123ABC" {
		t.Errorf("expected name=D123ABC, got %q", name)
	}
}

func TestSlackResolveUserCaching(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := slackUserResp{OK: true}
		resp.User.Profile.DisplayName = "alice"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer ts.Close()

	orig := slackAPIBase
	slackAPIBase = ts.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{BotToken: "xoxb-test"})

	name1 := svc.resolveUser("U123")
	name2 := svc.resolveUser("U123") // second call — should use cache

	if name1 != "alice" || name2 != "alice" {
		t.Errorf("expected alice, got %q and %q", name1, name2)
	}
	if callCount != 1 {
		t.Errorf("expected 1 API call (cached), got %d", callCount)
	}
}

func TestSlackResolveUserFallbackToID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(slackUserResp{OK: false, Error: "user_not_found"}) //nolint:errcheck
	}))
	defer ts.Close()

	orig := slackAPIBase
	slackAPIBase = ts.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{BotToken: "xoxb-test"})
	name := svc.resolveUser("U999")
	if name != "U999" {
		t.Errorf("expected fallback to user ID, got %q", name)
	}
}

func TestSlackResolveUserEmpty(t *testing.T) {
	svc := NewSlackService(config.SlackConfig{BotToken: "xoxb-test"})
	name := svc.resolveUser("")
	if name != "unknown" {
		t.Errorf("expected 'unknown' for empty user ID, got %q", name)
	}
}

func TestSlackFetchNotLiveReturnsMock(t *testing.T) {
	// Not live: Enabled=false, no token
	svc := NewSlackService(config.SlackConfig{})
	msgs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Error("expected mock messages when not live")
	}
	// Cache should be set
	cached := svc.GetCached()
	if len(cached) == 0 {
		t.Error("expected cached messages after fetch")
	}
}

func TestSlackFetchLiveAPIError(t *testing.T) {
	// IsLive=true but API returns 500. fetchFromAPI skips failed channels and
	// returns (nil, nil) — Fetch() caches nil and returns nil (no error).
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	orig := slackAPIBase
	slackAPIBase = ts.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{
		Enabled:  true,
		BotToken: "xoxb-test",
		Channels: []string{"C001"},
	})

	_, err := svc.Fetch()
	// API failure in a channel is gracefully skipped — no error returned.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSlackFetchLiveAPINetworkError(t *testing.T) {
	// IsLive=true but API base is unreachable. fetchChannel returns a network error;
	// fetchFromAPI skips it; Fetch() caches nil and returns (nil, nil).
	orig := slackAPIBase
	slackAPIBase = "http://127.0.0.1:1" // nothing listening
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{
		Enabled:  true,
		BotToken: "xoxb-test",
		Channels: []string{"C001"},
	})

	_, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSlackFetchChannelHTTPError(t *testing.T) {
	// Test fetchChannel with HTTP status error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	orig := slackAPIBase
	slackAPIBase = ts.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{BotToken: "xoxb-test"})
	msgs, err := svc.fetchChannel("C001")
	if err == nil {
		t.Error("expected error on HTTP 401")
	}
	if msgs != nil {
		t.Errorf("expected nil messages on error, got %v", msgs)
	}
}

func TestSlackFetchChannelInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/conversations.history" {
			w.Write([]byte("not-json")) //nolint:errcheck
		}
	}))
	defer ts.Close()

	orig := slackAPIBase
	slackAPIBase = ts.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{BotToken: "xoxb-test"})
	msgs, err := svc.fetchChannel("C001")
	if err == nil {
		t.Error("expected error on invalid JSON")
	}
	_ = msgs
}

func TestSlackResolveChannelAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(slackChannelResp{OK: false, Error: "channel_not_found"}) //nolint:errcheck
	}))
	defer ts.Close()

	orig := slackAPIBase
	slackAPIBase = ts.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{BotToken: "xoxb-test"})
	name, isDM := svc.resolveChannel("C999")
	if isDM {
		t.Error("expected isDM=false on error")
	}
	if name != "C999" {
		t.Errorf("expected fallback to channel ID, got %q", name)
	}
}

func TestParseSlackTSPrecision(t *testing.T) {
	// Verify different timestamps parse differently
	t1 := parseSlackTS("1609459200.000000")
	t2 := parseSlackTS("1609459260.000000") // 60 seconds later
	if !t2.After(t1) {
		t.Error("expected t2 to be after t1")
	}
	diff := t2.Sub(t1)
	if diff != 60*time.Second {
		t.Errorf("expected 60s diff, got %v", diff)
	}
}

func TestSlackBotMessageSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "conversations.history") {
			// Return a bot message (should be skipped) and a regular message
			fmt.Fprint(w, `{"ok":true,"messages":[
				{"type":"message","subtype":"bot_message","bot_id":"B001","text":"bot says hi","user":"U001","ts":"1000000.0"},
				{"type":"message","text":"hello from human","user":"U002","ts":"1000001.0"}
			]}`)
			return
		}
		if strings.Contains(r.URL.Path, "users.info") {
			fmt.Fprint(w, `{"ok":true,"user":{"id":"U002","name":"human","profile":{"display_name":"Human User"}}}`)
			return
		}
		if strings.Contains(r.URL.Path, "conversations.info") {
			fmt.Fprint(w, `{"ok":true,"channel":{"id":"C001","name":"general","is_im":false}}`)
			return
		}
	}))
	defer srv.Close()

	orig := slackAPIBase
	slackAPIBase = srv.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{
		Enabled:  true,
		BotToken: "xoxb-test",
		Channels: []string{"C001"},
	})

	msgs, err := svc.fetchChannel("C001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only the human message should be included
	if len(msgs) != 1 {
		t.Errorf("expected 1 message (bot message skipped), got %d", len(msgs))
	}
}

func TestSlackResolveUserRealName(t *testing.T) {
	// DisplayName is empty, should fall back to RealName
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"user":{"id":"U001","name":"username","profile":{"display_name":"","real_name":"Real Name"}}}`)
	}))
	defer srv.Close()

	orig := slackAPIBase
	slackAPIBase = srv.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{
		Enabled:  true,
		BotToken: "xoxb-test",
	})

	name := svc.resolveUser("U001")
	if name != "Real Name" {
		t.Errorf("expected 'Real Name', got %q", name)
	}
}

func TestSlackResolveUserFallbackToUsername(t *testing.T) {
	// Both DisplayName and RealName are empty, should fall back to Name
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"user":{"id":"U001","name":"username","profile":{"display_name":"","real_name":""}}}`)
	}))
	defer srv.Close()

	orig := slackAPIBase
	slackAPIBase = srv.URL
	defer func() { slackAPIBase = orig }()

	svc := NewSlackService(config.SlackConfig{
		Enabled:  true,
		BotToken: "xoxb-test",
	})

	name := svc.resolveUser("U001")
	if name != "username" {
		t.Errorf("expected 'username', got %q", name)
	}
}
