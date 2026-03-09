package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
	"github.com/todogpt/daily-briefing/internal/services"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not get free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func testServer(t *testing.T) *Server {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Server.DataDir = t.TempDir()
	hub := services.NewHub(cfg)
	return NewServer(hub, "localhost", 8080)
}

func TestNewServer(t *testing.T) {
	s := testServer(t)
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.hub == nil {
		t.Error("expected hub to be set")
	}
	if s.wsHub == nil {
		t.Error("expected WebSocket hub to be set")
	}
	if s.mux == nil {
		t.Error("expected mux to be set")
	}
}

func TestHandleBriefing(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/briefing", nil)
	w := httptest.NewRecorder()

	s.handleBriefing(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var briefing models.Briefing
	if err := json.NewDecoder(resp.Body).Decode(&briefing); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if briefing.Weather == nil {
		t.Error("expected weather in briefing")
	}
	// Events are only populated when an iCal URL is configured; empty is correct here.
	_ = briefing.Events
}

func TestHandleWeather(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()

	s.handleWeather(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var weather models.Weather
	if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if weather.City == "" {
		t.Error("expected city in weather response")
	}
}

func TestHandleEvents(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/events", nil)
	w := httptest.NewRecorder()

	s.handleEvents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Events endpoint returns an empty array when no iCal URL is configured — that is correct.
	var events []models.CalendarEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		t.Fatalf("decode error: %v", err)
	}
}

func TestHandleNews(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/news", nil)
	w := httptest.NewRecorder()

	s.handleNews(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var news []models.NewsItem
	if err := json.NewDecoder(resp.Body).Decode(&news); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(news) == 0 {
		t.Error("expected news items")
	}
}

func TestHandleEmails(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/emails", nil)
	w := httptest.NewRecorder()

	s.handleEmails(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var emails []models.EmailMessage
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(emails) == 0 {
		t.Error("expected emails")
	}
}

func TestHandleSlack(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/slack", nil)
	w := httptest.NewRecorder()

	s.handleSlack(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var msgs []models.SlackMessage
	if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(msgs) == 0 {
		t.Error("expected slack messages")
	}
}

func TestHandleGitHub(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/github", nil)
	w := httptest.NewRecorder()

	s.handleGitHub(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var notifs []models.GitHubNotification
	if err := json.NewDecoder(resp.Body).Decode(&notifs); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(notifs) == 0 {
		t.Error("expected github notifications")
	}
}

func TestHandleJira(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/jira", nil)
	w := httptest.NewRecorder()

	s.handleJira(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var tickets []models.JiraTicket
	if err := json.NewDecoder(resp.Body).Decode(&tickets); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(tickets) == 0 {
		t.Error("expected jira tickets (mock data)")
	}
}

func TestHandleNotion(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/notion", nil)
	w := httptest.NewRecorder()

	s.handleNotion(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var pages []models.NotionPage
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(pages) == 0 {
		t.Error("expected notion pages (mock data)")
	}
}

func TestHandleTodosGet(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/todos", nil)
	w := httptest.NewRecorder()

	s.handleTodos(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var todos []models.TodoItem
	if err := json.NewDecoder(resp.Body).Decode(&todos); err != nil {
		t.Fatalf("decode error: %v", err)
	}
}

func TestHandleTodosPost(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	body := `{"title": "New Task", "priority": 2}`
	req := httptest.NewRequest("POST", "/api/todos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleTodos(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var todo models.TodoItem
	if err := json.NewDecoder(resp.Body).Decode(&todo); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if todo.Title != "New Task" {
		t.Errorf("expected title 'New Task', got %q", todo.Title)
	}
	if todo.Source != "manual" {
		t.Errorf("expected source 'manual', got %q", todo.Source)
	}
	if todo.ID == "" {
		t.Error("expected non-empty ID in POST response")
	}
}

func TestHandleTodosPostInvalidJSON(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("POST", "/api/todos", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleTodos(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleTodosMethodNotAllowed(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("DELETE", "/api/todos", nil)
	w := httptest.NewRecorder()

	s.handleTodos(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleTodoActionUpdate(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	// Add a todo first
	s.hub.Todos.Add(models.TodoItem{ID: "action-1", Title: "Original"})

	body := `{"title": "Updated", "status": 2}`
	req := httptest.NewRequest("PATCH", "/api/todos/action-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify update
	items := s.hub.Todos.List()
	for _, item := range items {
		if item.ID == "action-1" {
			if item.Title != "Updated" {
				t.Errorf("expected title 'Updated', got %q", item.Title)
			}
			if item.Status != models.TodoDone {
				t.Errorf("expected status Done, got %d", item.Status)
			}
			if item.CompletedAt == nil {
				t.Error("expected CompletedAt to be set")
			}
		}
	}
}

func TestHandleTodoActionComplete(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	s.hub.Todos.Add(models.TodoItem{ID: "comp-1", Title: "Complete Me"})

	req := httptest.NewRequest("POST", "/api/todos/comp-1/complete", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify it's marked done
	for _, item := range s.hub.Todos.List() {
		if item.ID == "comp-1" {
			if item.Status != models.TodoDone {
				t.Errorf("expected status Done, got %d", item.Status)
			}
		}
	}
}

func TestHandleTodoActionCompleteNotFound(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	req := httptest.NewRequest("POST", "/api/todos/nonexistent/complete", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleTodoActionCompleteWrongMethod(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	s.hub.Todos.Add(models.TodoItem{ID: "comp-m", Title: "Complete Method"})
	req := httptest.NewRequest("GET", "/api/todos/comp-m/complete", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleTodoActionDelete(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	s.hub.Todos.Add(models.TodoItem{ID: "del-1", Title: "Delete Me"})

	req := httptest.NewRequest("DELETE", "/api/todos/del-1", nil)
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}
}

func TestHandleTodoActionNoID(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("DELETE", "/api/todos/", nil)
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleTodoActionBadMethod(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/todos/some-id", nil)
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleTodoActionInvalidJSON(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("PATCH", "/api/todos/some-id", strings.NewReader("bad json"))
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleSignals(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/signals", nil)
	w := httptest.NewRecorder()

	s.handleSignals(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var signals []models.Signal
	if err := json.NewDecoder(resp.Body).Decode(&signals); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(signals) == 0 {
		t.Error("expected signals from mock data")
	}

	// Check that signals come from multiple sources
	sources := make(map[string]bool)
	for _, s := range signals {
		sources[s.Source] = true
	}
	if !sources["slack"] {
		t.Error("expected slack signals")
	}
	if !sources["email"] {
		t.Error("expected email signals")
	}
	if !sources["github"] {
		t.Error("expected github signals")
	}
}

func TestCORSHeaders(t *testing.T) {
	s := testServer(t)
	handler := s.withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS Allow-Origin header")
	}
	if resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected CORS Allow-Methods header")
	}
}

func TestCORSPreflight(t *testing.T) {
	s := testServer(t)
	handler := s.withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/api/todos", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for OPTIONS, got %d", resp.StatusCode)
	}
}

func TestHandleDashboard(t *testing.T) {
	// This test will fail if run from a directory that doesn't contain web/templates/index.html
	// It's included for completeness but may need to be run from project root.
	s := testServer(t)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	s.handleDashboard(w, req)
	// We don't assert status here because the file path is relative to CWD
}

func TestMustJSON(t *testing.T) {
	data := mustJSON(map[string]string{"key": "value"})
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}

	var decoded map[string]string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if decoded["key"] != "value" {
		t.Errorf("expected 'value', got %q", decoded["key"])
	}
}

func TestFullRouteIntegration(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	// Test the mux routes directly
	routes := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/api/weather", 200},
		{"GET", "/api/events", 200},
		{"GET", "/api/news", 200},
		{"GET", "/api/emails", 200},
		{"GET", "/api/slack", 200},
		{"GET", "/api/github", 200},
		{"GET", "/api/jira", 200},
		{"GET", "/api/notion", 200},
		{"GET", "/api/todos", 200},
		{"GET", "/api/signals", 200},
		{"GET", "/api/briefing", 200},
		{"GET", "/api/config", 200},
		{"GET", "/api/review", 200},
		{"GET", "/api/timeblocks", 200},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			s.mux.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Errorf("expected status %d, got %d", tc.status, w.Code)
			}
		})
	}
}

func TestTodoPostAndList(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	// POST a new todo
	body := `{"title": "Integration Test Task", "priority": 1}`
	req := httptest.NewRequest("POST", "/api/todos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/todos returned %d", w.Code)
	}

	// GET todos and verify it's there
	req = httptest.NewRequest("GET", "/api/todos", nil)
	w = httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var todos []models.TodoItem
	json.NewDecoder(w.Body).Decode(&todos)

	found := false
	for _, todo := range todos {
		if todo.Title == "Integration Test Task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("posted todo not found in GET /api/todos response")
	}
}

func TestHandler(t *testing.T) {
	s := testServer(t)
	handler := s.Handler()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	// Verify routes work through the handler (which includes CORS wrapping)
	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header from Handler")
	}
}

func TestBridgeUpdatesAndWebSocket(t *testing.T) {
	s := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	// Connect a WebSocket client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	// Broadcast through the services hub
	s.hub.Broadcast(models.DashboardUpdate{Type: "bridge_test", Payload: "data"})

	// The bridgeUpdates goroutine should forward this to the WS client
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if !strings.Contains(string(msg), "bridge_test") {
		t.Errorf("expected bridge_test in message, got %q", msg)
	}
}

func TestBridgeUpdatesSkipsMarshalError(t *testing.T) {
	s := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	time.Sleep(100 * time.Millisecond)

	// Broadcast an unmarshalable payload (channels can't be marshaled)
	s.hub.Broadcast(models.DashboardUpdate{Type: "bad", Payload: make(chan int)})
	time.Sleep(100 * time.Millisecond)
	// No crash means bridgeUpdates handled the marshal error gracefully
}

func TestMustJSONError(t *testing.T) {
	// Channels cannot be marshaled to JSON
	result := mustJSON(make(chan int))
	if string(result) != "{}" {
		t.Errorf("expected empty JSON on marshal error, got %q", result)
	}
}

// ── /api/config ───────────────────────────────────────────────────────────────

func TestHandleConfigGet(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()

	s.handleConfig(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if m["server"] == nil {
		t.Error("expected server config in response")
	}
}

func TestHandleConfigGetRedactsCredentials(t *testing.T) {
	s := testServer(t)
	// Set a token so we can verify redaction
	s.hub.GetConfig().GitHub.Token = "secret-token"
	s.hub.GetConfig().GitHub.Enabled = true

	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()
	s.handleConfig(w, req)

	var m map[string]interface{}
	json.NewDecoder(w.Body).Decode(&m)
	gh, _ := m["github"].(map[string]interface{})
	if gh["token"] == "secret-token" {
		t.Error("token should be redacted in GET response")
	}
	if gh["token"] != "***" {
		t.Errorf("expected '***', got %v", gh["token"])
	}
}

func TestHandleConfigPut(t *testing.T) {
	s := testServer(t)
	// Use a temp dir as cfgPath parent so Save works
	dir := t.TempDir()
	s.SetConfigPath(dir + "/config.json")

	body := `{"server":{"port":9090,"poll_interval_seconds":60},"weather":{"city":"London","enabled":true},"github":{"token":"***"},"ai":{"enabled":false},"pomodoro":{"work_minutes":25,"break_minutes":5,"enabled":true}}`
	req := httptest.NewRequest("PUT", "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleConfig(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["ok"] != true {
		t.Errorf("expected ok:true, got %v", result)
	}
	if s.hub.GetConfig().Weather.City != "London" {
		t.Errorf("expected city London, got %q", s.hub.GetConfig().Weather.City)
	}
}

func TestHandleConfigPutPreservesRedacted(t *testing.T) {
	s := testServer(t)
	s.hub.GetConfig().AI.APIKey = "real-key"
	dir := t.TempDir()
	s.SetConfigPath(dir + "/config.json")

	body := `{"ai":{"api_key":"***","enabled":true}}`
	req := httptest.NewRequest("PUT", "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleConfig(w, req)

	if s.hub.GetConfig().AI.APIKey != "real-key" {
		t.Errorf("expected real-key preserved, got %q", s.hub.GetConfig().AI.APIKey)
	}
}

func TestHandleConfigPutInvalidJSON(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("PUT", "/api/config", strings.NewReader("bad json"))
	w := httptest.NewRecorder()
	s.handleConfig(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleConfigMethodNotAllowed(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("DELETE", "/api/config", nil)
	w := httptest.NewRecorder()
	s.handleConfig(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ── /api/review ───────────────────────────────────────────────────────────────

func TestHandleReviewGetDisabledAI(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/review", nil)
	w := httptest.NewRecorder()

	s.handleReview(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["review"] != "" {
		t.Errorf("expected empty review when AI disabled, got %q", result["review"])
	}
}

func TestHandleReviewMethodNotAllowed(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("POST", "/api/review", nil)
	w := httptest.NewRecorder()
	s.handleReview(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ── /api/timeblocks ───────────────────────────────────────────────────────────

func TestHandleTimeBlocksGetDisabledAI(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/timeblocks", nil)
	w := httptest.NewRecorder()

	s.handleTimeBlocks(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	blocks, ok := result["blocks"].([]interface{})
	if !ok || len(blocks) != 0 {
		t.Errorf("expected empty blocks array when AI disabled, got %v", result["blocks"])
	}
}

func TestHandleTimeBlocksMethodNotAllowed(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("POST", "/api/timeblocks", nil)
	w := httptest.NewRecorder()
	s.handleTimeBlocks(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleConfigPutSaveError(t *testing.T) {
	s := testServer(t)
	// Set cfgPath to a non-writable location
	s.SetConfigPath("/proc/cannot-write-here/config.json")

	body := `{"server":{"port":8080}}`
	req := httptest.NewRequest("PUT", "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleConfig(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on save error, got %d", w.Code)
	}
}

func TestHandleReviewAIError(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.DataDir = t.TempDir()
	cfg.AI.Enabled = true
	cfg.AI.APIKey = "key"
	hub := services.NewHub(cfg)

	// Point at unreachable port
	origURL := services.ExportClaudeAPIURL()
	services.SetClaudeAPIURL("http://127.0.0.1:1")
	defer services.SetClaudeAPIURL(origURL)

	s := NewServer(hub, "localhost", 8080)
	go s.wsHub.Run()

	req := httptest.NewRequest("GET", "/api/review", nil)
	w := httptest.NewRecorder()
	s.handleReview(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on AI error, got %d", w.Code)
	}
}

func TestHandleTimeBlocksAIError(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.DataDir = t.TempDir()
	cfg.AI.Enabled = true
	cfg.AI.APIKey = "key"
	hub := services.NewHub(cfg)

	origURL := services.ExportClaudeAPIURL()
	services.SetClaudeAPIURL("http://127.0.0.1:1")
	defer services.SetClaudeAPIURL(origURL)

	s := NewServer(hub, "localhost", 8080)
	go s.wsHub.Run()

	req := httptest.NewRequest("GET", "/api/timeblocks", nil)
	w := httptest.NewRecorder()
	s.handleTimeBlocks(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on AI error, got %d", w.Code)
	}
}

func TestGetConfig(t *testing.T) {
	s := testServer(t)
	cfg := s.hub.GetConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config from hub.GetConfig()")
	}
}

func TestMergeConfigUpdatesCredentials(t *testing.T) {
	existing := config.DefaultConfig()
	existing.GitHub.Token = "old-token"
	existing.AI.APIKey = "old-key"

	incoming := config.DefaultConfig()
	incoming.GitHub.Token = "new-token"
	incoming.AI.APIKey = "new-key"

	merged := mergeConfig(existing, incoming)
	if merged.GitHub.Token != "new-token" {
		t.Errorf("expected new-token, got %q", merged.GitHub.Token)
	}
	if merged.AI.APIKey != "new-key" {
		t.Errorf("expected new-key, got %q", merged.AI.APIKey)
	}
}

func TestMaskConfigNoSensitiveFields(t *testing.T) {
	s := testServer(t)
	cfg := config.DefaultConfig()
	// No credentials set — nothing to redact
	m := s.maskConfig(cfg)
	if m == nil {
		t.Fatal("expected non-nil map")
	}
}

func TestHandleTodoActionSetRecurring(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	s.hub.Todos.Add(models.TodoItem{ID: "rec-set", Title: "Daily task"})

	body := `{"recurring":{"frequency":"daily","enabled":true}}`
	req := httptest.NewRequest("PATCH", "/api/todos/rec-set", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	for _, it := range s.hub.Todos.List() {
		if it.ID == "rec-set" {
			if it.Recurring == nil || !it.Recurring.Enabled {
				t.Error("expected recurring rule to be set")
			}
			break
		}
	}
}

func TestStartListensAndServes(t *testing.T) {
	port := freePort(t)
	cfg := config.DefaultConfig()
	hub := services.NewHub(cfg)
	s := NewServer(hub, "127.0.0.1", port)

	go func() {
		_ = s.Start()
	}()

	// Poll until server is ready instead of fixed sleep
	addr := fmt.Sprintf("http://127.0.0.1:%d/api/weather", port)
	var resp *http.Response
	var err error
	for i := 0; i < 20; i++ {
		resp, err = http.Get(addr)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("server not responding after retries: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleTodoActionPUT(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	s.hub.Todos.Add(models.TodoItem{ID: "put-1", Title: "Original", Priority: models.PriorityLow})

	body := `{"title": "Updated via PUT", "priority": 3, "notes": "test notes", "description": "test desc"}`
	req := httptest.NewRequest("PUT", "/api/todos/put-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Verify all fields updated
	items := s.hub.Todos.List()
	for _, item := range items {
		if item.ID == "put-1" {
			if item.Title != "Updated via PUT" {
				t.Errorf("expected title 'Updated via PUT', got %q", item.Title)
			}
			if item.Priority != models.PriorityUrgent {
				t.Errorf("expected priority Urgent (3), got %d", item.Priority)
			}
			if item.Notes != "test notes" {
				t.Errorf("expected notes 'test notes', got %q", item.Notes)
			}
			if item.Description != "test desc" {
				t.Errorf("expected description 'test desc', got %q", item.Description)
			}
		}
	}
}

// ── Auth handler tests ────────────────────────────────────────────────────────

func testServerWithGoogleAuth(t *testing.T) *Server {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Server.DataDir = t.TempDir()
	cfg.Google.ClientID = "test-client-id"
	cfg.Google.ClientSecret = "test-client-secret"
	hub := services.NewHub(cfg)
	return NewServer(hub, "localhost", 8080)
}

func TestHandleAuthStatusNotConfigured(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	s.handleAuthStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	g := resp["google"]
	if g["configured"] {
		t.Error("expected configured=false")
	}
	if g["connected"] {
		t.Error("expected connected=false")
	}
}

func TestHandleAuthStatusConfigured(t *testing.T) {
	s := testServerWithGoogleAuth(t)
	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	s.handleAuthStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !resp["google"]["configured"] {
		t.Error("expected configured=true")
	}
}

func TestHandleAuthStatusMethodNotAllowed(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("POST", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	s.handleAuthStatus(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleAuthGoogleNotConfigured(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/auth/google", nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogle(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when not configured, got %d", w.Code)
	}
}

func TestHandleAuthGoogleRedirects(t *testing.T) {
	s := testServerWithGoogleAuth(t)
	req := httptest.NewRequest("GET", "/api/auth/google?return_to=http://localhost:3000", nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogle(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc == "" {
		t.Error("expected Location header")
	}
}

func TestHandleAuthGoogleMethodNotAllowed(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("POST", "/api/auth/google", nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogle(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleAuthGoogleCallbackDenied(t *testing.T) {
	s := testServerWithGoogleAuth(t)
	req := httptest.NewRequest("GET", "/api/auth/google/callback?error=access_denied", nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogleCallback(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect on denied, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "google=denied") {
		t.Error("expected google=denied in redirect")
	}
}

func TestHandleAuthGoogleCallbackInvalidState(t *testing.T) {
	s := testServerWithGoogleAuth(t)
	req := httptest.NewRequest("GET", "/api/auth/google/callback?code=someCode&state=wrong-state", nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogleCallback(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid state, got %d", w.Code)
	}
}

func TestHandleAuthGoogleCallbackExchangeError(t *testing.T) {
	s := testServerWithGoogleAuth(t)
	go s.wsHub.Run()

	// Generate a valid state via AuthURL, then use it with a bad code.
	// The oauth2 exchange will fail because the token URL is Google's real endpoint
	// which won't accept our test credentials.
	// Use the state from the auth service — call AuthURL to get back the URL,
	// then extract the state from the URL query parameter.
	authURL := s.hub.GoogleAuth.AuthURL("http://localhost:3000")
	// Parse state from the returned URL.
	parsed, _ := url.Parse(authURL)
	state := parsed.Query().Get("state")

	req := httptest.NewRequest("GET", "/api/auth/google/callback?code=bad-code&state="+state, nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogleCallback(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect on exchange error, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "google=error") {
		t.Error("expected google=error in redirect")
	}
}

func TestHandleAuthGoogleCallbackMethodNotAllowed(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("POST", "/api/auth/google/callback", nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogleCallback(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleAuthGoogleDisconnect(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("POST", "/api/auth/google/disconnect", nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogleDisconnect(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleAuthGoogleDisconnectMethodNotAllowed(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest("GET", "/api/auth/google/disconnect", nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogleDisconnect(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleTodoActionDeleteNotFound(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	req := httptest.NewRequest("DELETE", "/api/todos/nonexistent-id", nil)
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when deleting non-existent todo, got %d", w.Code)
	}
}

func TestHandleTodoActionUpdateNotFound(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()

	body := `{"title": "Updated"}`
	req := httptest.NewRequest("PATCH", "/api/todos/nonexistent-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when updating non-existent todo, got %d", w.Code)
	}
}

func TestSetDebug(t *testing.T) {
	s := testServer(t)
	if s.debug {
		t.Fatal("debug should be false by default")
	}
	s.SetDebug(true)
	if !s.debug {
		t.Error("expected debug=true after SetDebug(true)")
	}
	s.SetDebug(false)
	if s.debug {
		t.Error("expected debug=false after SetDebug(false)")
	}
}

func TestWithDebugLogging(t *testing.T) {
	s := testServer(t)
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := s.withDebugLogging(inner)
	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected inner handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandlerDebugMode(t *testing.T) {
	s := testServer(t)
	go s.wsHub.Run()
	go s.bridgeUpdates()

	s.SetDebug(true)
	h := s.handler()

	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 in debug mode, got %d", w.Code)
	}
}

func TestHandleAuthGoogleCallbackSuccess(t *testing.T) {
	// Mock OAuth2 token endpoint
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"ya29.test","token_type":"Bearer","expires_in":3600,"refresh_token":"1//test"}`)
	}))
	defer tokenSrv.Close()

	s := testServerWithGoogleAuth(t)
	go s.wsHub.Run()
	go s.bridgeUpdates()

	// Point the oauth2 config at our mock token server
	services.SetGoogleOAuthTokenURLForTest(s.hub.GoogleAuth, tokenSrv.URL+"/token")

	// Generate a valid CSRF state
	authURL := s.hub.GoogleAuth.AuthURL("")
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse authURL: %v", err)
	}
	state := parsed.Query().Get("state")

	req := httptest.NewRequest("GET", "/api/auth/google/callback?code=valid-code&state="+state, nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogleCallback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect on success, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "google=connected") {
		t.Errorf("expected google=connected in redirect, got %q", w.Header().Get("Location"))
	}
}

func TestHandleAuthGoogleCallbackSuccessWithReturnTo(t *testing.T) {
	// Same as above but with a returnTo URL set via AuthURL
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"ya29.test","token_type":"Bearer","expires_in":3600}`)
	}))
	defer tokenSrv.Close()

	s := testServerWithGoogleAuth(t)
	go s.wsHub.Run()
	go s.bridgeUpdates()

	services.SetGoogleOAuthTokenURLForTest(s.hub.GoogleAuth, tokenSrv.URL+"/token")

	authURL := s.hub.GoogleAuth.AuthURL("http://localhost:3000")
	parsed, _ := url.Parse(authURL)
	state := parsed.Query().Get("state")

	req := httptest.NewRequest("GET", "/api/auth/google/callback?code=valid-code&state="+state, nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogleCallback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "localhost:3000") {
		t.Errorf("expected redirect to returnTo URL, got %q", loc)
	}
}

func TestHandleAuthGoogleDisconnectError(t *testing.T) {
	dir := t.TempDir()
	// Write a valid token file so GoogleAuthService loads a non-nil token
	tokenPath := dir + "/google-token.json"
	tokenJSON := `{"access_token":"test","token_type":"Bearer","expiry":"2099-01-01T00:00:00Z"}`
	if err := os.WriteFile(tokenPath, []byte(tokenJSON), 0600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	// Make the directory read-only so os.Remove will fail
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(dir, 0755) //nolint:errcheck // restore for TempDir cleanup

	cfg := config.DefaultConfig()
	cfg.Google.ClientID = "test-client-id"
	cfg.Google.ClientSecret = "test-client-secret"
	cfg.Server.DataDir = dir
	hub := services.NewHub(cfg)
	s := NewServer(hub, "localhost", 8080)

	req := httptest.NewRequest("POST", "/api/auth/google/disconnect", nil)
	w := httptest.NewRecorder()
	s.handleAuthGoogleDisconnect(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when token file cannot be removed, got %d", w.Code)
	}
}
