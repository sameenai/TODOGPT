package services

import (
	"os"
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

func testConfig() *config.Config {
	cfg := config.DefaultConfig()
	// Use a temp dir so tests are isolated from ~/.daily-briefing/todos.json
	cfg.Server.DataDir = os.TempDir() + "/todogpt-test-hub"
	return cfg
}

func TestNewHub(t *testing.T) {
	hub := NewHub(testConfig())
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
	if hub.Weather == nil {
		t.Error("expected Weather service")
	}
	if hub.News == nil {
		t.Error("expected News service")
	}
	if hub.Calendar == nil {
		t.Error("expected Calendar service")
	}
	if hub.Slack == nil {
		t.Error("expected Slack service")
	}
	if hub.Email == nil {
		t.Error("expected Email service")
	}
	if hub.GitHub == nil {
		t.Error("expected GitHub service")
	}
	if hub.Todos == nil {
		t.Error("expected Todos service")
	}
}

func TestHubFetchAll(t *testing.T) {
	hub := NewHub(testConfig())
	briefing := hub.FetchAll()

	if briefing == nil {
		t.Fatal("expected non-nil briefing")
	}
	if briefing.Weather == nil {
		t.Error("expected weather data")
	}
	// Calendar events are only present when an iCal URL is configured; no URL = empty.
	_ = briefing.Events
	if len(briefing.News) == 0 {
		t.Error("expected news items")
	}
	if len(briefing.UnreadEmails) == 0 {
		t.Error("expected emails")
	}
	if len(briefing.SlackMessages) == 0 {
		t.Error("expected slack messages")
	}
	if len(briefing.GitHubNotifs) == 0 {
		t.Error("expected github notifications")
	}
	if briefing.GeneratedAt.IsZero() {
		t.Error("expected GeneratedAt to be set")
	}
	if briefing.Date.IsZero() {
		t.Error("expected Date to be set")
	}
}

func TestHubFetchAllGeneratesTodos(t *testing.T) {
	hub := NewHub(testConfig())
	briefing := hub.FetchAll()

	if len(briefing.Todos) == 0 {
		t.Error("expected auto-generated todos from briefing signals")
	}

	// Verify todos have required fields
	for _, todo := range briefing.Todos {
		if todo.ID == "" {
			t.Error("todo should have an ID")
		}
		if todo.Title == "" {
			t.Error("todo should have a title")
		}
		if todo.Source == "" {
			t.Error("todo should have a source")
		}
	}
}

func TestHubSubscribeAndBroadcast(t *testing.T) {
	hub := NewHub(testConfig())
	ch := hub.Subscribe()

	update := models.DashboardUpdate{
		Type:    "test_update",
		Payload: "hello",
	}

	go func() {
		hub.Broadcast(update)
	}()

	select {
	case received := <-ch:
		if received.Type != "test_update" {
			t.Errorf("expected type 'test_update', got %q", received.Type)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for broadcast")
	}
}

func TestHubMultipleSubscribers(t *testing.T) {
	hub := NewHub(testConfig())
	ch1 := hub.Subscribe()
	ch2 := hub.Subscribe()
	ch3 := hub.Subscribe()

	update := models.DashboardUpdate{Type: "multi_test", Payload: nil}
	go func() {
		hub.Broadcast(update)
	}()

	for i, ch := range []chan models.DashboardUpdate{ch1, ch2, ch3} {
		select {
		case received := <-ch:
			if received.Type != "multi_test" {
				t.Errorf("subscriber %d: expected type 'multi_test', got %q", i, received.Type)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("subscriber %d: timeout", i)
		}
	}
}

func TestHubUnsubscribe(t *testing.T) {
	hub := NewHub(testConfig())
	ch := hub.Subscribe()

	hub.Unsubscribe(ch)

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

func TestHubUnsubscribeNonExistent(t *testing.T) {
	hub := NewHub(testConfig())
	fakeCh := make(chan models.DashboardUpdate)
	// Should not panic
	hub.Unsubscribe(fakeCh)
}

func TestHubStartAndStop(t *testing.T) {
	cfg := testConfig()
	cfg.Server.PollInterval = 1 // 1 second for test

	hub := NewHub(cfg)
	ch := hub.Subscribe()

	go hub.StartPolling()

	// Wait for initial broadcast
	select {
	case update := <-ch:
		if update.Type != "full_refresh" {
			t.Errorf("expected type 'full_refresh', got %q", update.Type)
		}
	case <-time.After(3 * time.Second):
		t.Error("timeout waiting for initial polling broadcast")
	}

	hub.Stop()
}

func TestHubBroadcastDropsSlowListeners(t *testing.T) {
	hub := NewHub(testConfig())
	ch := hub.Subscribe()

	// Fill the channel buffer (50) + 1 to trigger drop
	for i := 0; i < 55; i++ {
		hub.Broadcast(models.DashboardUpdate{Type: "flood", Payload: i})
	}

	// Channel should still have messages up to buffer size
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count == 0 {
		t.Error("expected some messages to be received")
	}
}

func TestHubEmailCount(t *testing.T) {
	hub := NewHub(testConfig())
	briefing := hub.FetchAll()

	if briefing.EmailCount == 0 {
		t.Error("expected non-zero email count")
	}
}

func TestHubSlackUnread(t *testing.T) {
	hub := NewHub(testConfig())
	briefing := hub.FetchAll()

	if briefing.SlackUnread == 0 {
		t.Error("expected non-zero slack unread count")
	}
}
