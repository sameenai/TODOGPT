package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/todogpt/daily-briefing/internal/api"
	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
	"github.com/todogpt/daily-briefing/internal/services"
)

// TestEndToEndBriefingFlow tests the complete lifecycle:
// config -> hub -> fetch -> API response -> WebSocket delivery
func TestEndToEndBriefingFlow(t *testing.T) {
	// 1. Load default config
	cfg := config.DefaultConfig()
	cfg.Server.PollInterval = 60 // don't poll during test

	// 2. Create hub and server
	hub := services.NewHub(cfg)
	server := api.NewServer(hub, "localhost", 0)
	_ = server // we test via handlers directly but validate creation

	// 3. Fetch briefing
	briefing := hub.FetchAll()

	if briefing == nil {
		t.Fatal("FetchAll returned nil")
	}

	// 4. Verify all sections are populated
	if briefing.Weather == nil {
		t.Error("missing weather")
	}
	// Calendar events require an iCal URL in config; empty is correct when unconfigured.
	_ = briefing.Events
	if len(briefing.News) == 0 {
		t.Error("missing news")
	}
	if len(briefing.UnreadEmails) == 0 {
		t.Error("missing emails")
	}
	if len(briefing.SlackMessages) == 0 {
		t.Error("missing slack messages")
	}
	if len(briefing.GitHubNotifs) == 0 {
		t.Error("missing github notifs")
	}

	// 5. Verify todos were auto-generated
	if len(briefing.Todos) == 0 {
		t.Error("expected auto-generated todos")
	}

	// 6. Verify email count
	if briefing.EmailCount == 0 {
		t.Error("expected non-zero email count")
	}

	// 7. Verify slack unread count
	if briefing.SlackUnread == 0 {
		t.Error("expected non-zero slack unread count")
	}
}

// TestTodoLifecycleIntegration tests creating, updating, completing, and deleting todos via the API.
func TestTodoLifecycleIntegration(t *testing.T) {
	cfg := config.DefaultConfig()
	hub := services.NewHub(cfg)
	srv := api.NewServer(hub, "localhost", 0)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. POST a new todo
	body := `{"title": "Integration Test Todo", "priority": 2}`
	resp, err := http.Post(ts.URL+"/api/todos", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/todos error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("POST /api/todos expected 201, got %d", resp.StatusCode)
	}

	var created models.TodoItem
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()

	if created.Title != "Integration Test Todo" {
		t.Errorf("expected title 'Integration Test Todo', got %q", created.Title)
	}
	if created.Source != "manual" {
		t.Errorf("expected source 'manual', got %q", created.Source)
	}

	// 2. GET todos and verify it exists
	resp, err = http.Get(ts.URL + "/api/todos")
	if err != nil {
		t.Fatalf("GET /api/todos error: %v", err)
	}
	var todos []models.TodoItem
	json.NewDecoder(resp.Body).Decode(&todos)
	resp.Body.Close()

	found := false
	for _, todo := range todos {
		if todo.Title == "Integration Test Todo" {
			found = true
		}
	}
	if !found {
		t.Error("created todo not found in list")
	}

	// 3. PATCH to complete
	todoID := created.ID
	if todoID == "" {
		// Find the ID from the list
		for _, todo := range todos {
			if todo.Title == "Integration Test Todo" {
				todoID = todo.ID
				break
			}
		}
	}

	if todoID != "" {
		patchBody := `{"status": 2}`
		req, _ := http.NewRequest("PATCH", ts.URL+"/api/todos/"+todoID, strings.NewReader(patchBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("PATCH error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("PATCH expected 200, got %d", resp.StatusCode)
		}
		resp.Body.Close()

		// 4. DELETE the todo
		req, _ = http.NewRequest("DELETE", ts.URL+"/api/todos/"+todoID, nil)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("DELETE error: %v", err)
		}
		if resp.StatusCode != 204 {
			t.Errorf("DELETE expected 204, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// TestWebSocketIntegration tests WebSocket connection and receiving real-time updates.
func TestWebSocketIntegration(t *testing.T) {
	cfg := config.DefaultConfig()
	hub := services.NewHub(cfg)
	srv := api.NewServer(hub, "localhost", 0)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Connect WebSocket
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial error: %v", err)
	}
	defer conn.Close()

	// POST a todo — should trigger WebSocket update
	body := `{"title": "WebSocket Test Todo"}`
	_, err = http.Post(ts.URL+"/api/todos", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST error: %v", err)
	}

	// Read WebSocket message
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("WebSocket read error: %v", err)
	}

	var update models.DashboardUpdate
	if err := json.Unmarshal(msg, &update); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if update.Type != "todos_updated" {
		t.Errorf("expected update type 'todos_updated', got %q", update.Type)
	}
}

// TestSignalsAggregation tests the unified signal feed endpoint.
func TestSignalsAggregation(t *testing.T) {
	cfg := config.DefaultConfig()
	hub := services.NewHub(cfg)
	srv := api.NewServer(hub, "localhost", 0)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/signals")
	if err != nil {
		t.Fatalf("GET /api/signals error: %v", err)
	}
	defer resp.Body.Close()

	var signals []models.Signal
	json.NewDecoder(resp.Body).Decode(&signals)

	if len(signals) == 0 {
		t.Error("expected signals from aggregated sources")
	}

	sources := make(map[string]int)
	for _, s := range signals {
		sources[s.Source]++
	}

	if sources["slack"] == 0 {
		t.Error("expected slack signals")
	}
	if sources["email"] == 0 {
		t.Error("expected email signals")
	}
	if sources["github"] == 0 {
		t.Error("expected github signals")
	}
}

// TestBriefingAPIResponse tests the full /api/briefing response structure.
func TestBriefingAPIResponse(t *testing.T) {
	cfg := config.DefaultConfig()
	hub := services.NewHub(cfg)
	srv := api.NewServer(hub, "localhost", 0)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/briefing")
	if err != nil {
		t.Fatalf("GET /api/briefing error: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}

	var briefing models.Briefing
	if err := json.NewDecoder(resp.Body).Decode(&briefing); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if briefing.GeneratedAt.IsZero() {
		t.Error("expected GeneratedAt to be set")
	}
}

// TestConfigSaveLoadIntegration tests config save and load with custom settings.
func TestConfigSaveLoadIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/test-config.json"

	cfg := config.DefaultConfig()
	cfg.Server.Port = 9999
	cfg.Weather.City = "Berlin"
	// No API key — will use mock data with the configured city

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Server.Port != 9999 {
		t.Errorf("expected port 9999, got %d", loaded.Server.Port)
	}
	if loaded.Weather.City != "Berlin" {
		t.Errorf("expected city Berlin from loaded config, got %s", loaded.Weather.City)
	}

	// Use loaded config to create a hub
	hub := services.NewHub(loaded)
	briefing := hub.FetchAll()

	if briefing.Weather.City != "Berlin" {
		t.Errorf("expected city Berlin from mock weather, got %s", briefing.Weather.City)
	}
}

// TestHubSubscriptionIntegration tests real-time updates through the hub.
func TestHubSubscriptionIntegration(t *testing.T) {
	cfg := config.DefaultConfig()
	hub := services.NewHub(cfg)

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	// Broadcast an update
	go func() {
		update := models.DashboardUpdate{
			Type:    "test_integration",
			Payload: "data",
		}
		hub.Broadcast(update)
	}()

	select {
	case received := <-ch:
		if received.Type != "test_integration" {
			t.Errorf("expected type test_integration, got %s", received.Type)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for broadcast")
	}
}
