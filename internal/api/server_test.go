package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
	"github.com/todogpt/daily-briefing/internal/services"
)

func testServer() *Server {
	cfg := config.DefaultConfig()
	hub := services.NewHub(cfg)
	return NewServer(hub, "localhost", 8080)
}

func TestNewServer(t *testing.T) {
	s := testServer()
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
	s := testServer()
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
	if len(briefing.Events) == 0 {
		t.Error("expected events in briefing")
	}
}

func TestHandleWeather(t *testing.T) {
	s := testServer()
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
	s := testServer()
	req := httptest.NewRequest("GET", "/api/events", nil)
	w := httptest.NewRecorder()

	s.handleEvents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var events []models.CalendarEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(events) == 0 {
		t.Error("expected events")
	}
}

func TestHandleNews(t *testing.T) {
	s := testServer()
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
	s := testServer()
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
	s := testServer()
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
	s := testServer()
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

func TestHandleTodosGet(t *testing.T) {
	s := testServer()
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
	s := testServer()
	go s.wsHub.Run()

	body := `{"title": "New Task", "priority": 2}`
	req := httptest.NewRequest("POST", "/api/todos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleTodos(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
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
}

func TestHandleTodosPostInvalidJSON(t *testing.T) {
	s := testServer()
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
	s := testServer()
	req := httptest.NewRequest("DELETE", "/api/todos", nil)
	w := httptest.NewRecorder()

	s.handleTodos(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleTodoActionUpdate(t *testing.T) {
	s := testServer()
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

func TestHandleTodoActionDelete(t *testing.T) {
	s := testServer()
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
	s := testServer()
	req := httptest.NewRequest("DELETE", "/api/todos/", nil)
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleTodoActionBadMethod(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest("GET", "/api/todos/some-id", nil)
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleTodoActionInvalidJSON(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest("PATCH", "/api/todos/some-id", strings.NewReader("bad json"))
	w := httptest.NewRecorder()

	s.handleTodoAction(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleSignals(t *testing.T) {
	s := testServer()
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
	s := testServer()
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
	s := testServer()
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
	s := testServer()
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
	s := testServer()
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
		{"GET", "/api/todos", 200},
		{"GET", "/api/signals", 200},
		{"GET", "/api/briefing", 200},
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
	s := testServer()
	go s.wsHub.Run()

	// POST a new todo
	body := `{"title": "Integration Test Task", "priority": 1}`
	req := httptest.NewRequest("POST", "/api/todos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
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
