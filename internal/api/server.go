package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
	"github.com/todogpt/daily-briefing/internal/services"
	"github.com/todogpt/daily-briefing/internal/websocket"
)

// configRedacted is the placeholder shown in GET /api/config for set credentials.
const configRedacted = "***"

type Server struct {
	hub     *services.Hub
	wsHub   *websocket.Hub
	mux     *http.ServeMux
	port    int
	host    string
	cfgPath string // path used when saving config; "" = default ~/.daily-briefing/config.json
	debug   bool
}

func NewServer(hub *services.Hub, host string, port int) *Server {
	s := &Server{
		hub:   hub,
		wsHub: websocket.NewHub(),
		mux:   http.NewServeMux(),
		port:  port,
		host:  host,
	}
	s.registerRoutes()
	return s
}

// SetConfigPath sets the config file path used by PUT /api/config.
func (s *Server) SetConfigPath(path string) {
	s.cfgPath = path
}

// SetDebug enables verbose request logging for local development.
func (s *Server) SetDebug(enabled bool) {
	s.debug = enabled
}

func (s *Server) registerRoutes() {
	// API routes
	s.mux.HandleFunc("/api/briefing", s.handleBriefing)
	s.mux.HandleFunc("/api/weather", s.handleWeather)
	s.mux.HandleFunc("/api/events", s.handleEvents)
	s.mux.HandleFunc("/api/news", s.handleNews)
	s.mux.HandleFunc("/api/emails", s.handleEmails)
	s.mux.HandleFunc("/api/slack", s.handleSlack)
	s.mux.HandleFunc("/api/github", s.handleGitHub)
	s.mux.HandleFunc("/api/jira", s.handleJira)
	s.mux.HandleFunc("/api/notion", s.handleNotion)
	s.mux.HandleFunc("/api/todos", s.handleTodos)
	s.mux.HandleFunc("/api/todos/", s.handleTodoAction)
	s.mux.HandleFunc("/api/signals", s.handleSignals)
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/review", s.handleReview)
	s.mux.HandleFunc("/api/timeblocks", s.handleTimeBlocks)
	s.mux.HandleFunc("/api/auth/status", s.handleAuthStatus)
	s.mux.HandleFunc("/api/auth/google", s.handleAuthGoogle)
	s.mux.HandleFunc("/api/auth/google/callback", s.handleAuthGoogleCallback)
	s.mux.HandleFunc("/api/auth/google/disconnect", s.handleAuthGoogleDisconnect)

	// WebSocket
	s.mux.HandleFunc("/ws", s.handleWebSocket)

	// Static files
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Dashboard
	s.mux.HandleFunc("/", s.handleDashboard)
}

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() http.Handler {
	go s.wsHub.Run()
	go s.bridgeUpdates()
	return s.withCORS(s.mux)
}

// handler returns the full middleware-wrapped handler for production use.
func (s *Server) handler() http.Handler {
	h := s.withCORS(s.mux)
	if s.debug {
		h = s.withDebugLogging(h)
	}
	return h
}

func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	// Bridge service hub updates to WebSocket clients
	go s.bridgeUpdates()

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	log.Printf("Dashboard running at http://%s", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return srv.ListenAndServe()
}

func (s *Server) bridgeUpdates() {
	ch := s.hub.Subscribe()
	for update := range ch {
		data, err := json.Marshal(update)
		if err != nil {
			continue
		}
		s.wsHub.Broadcast(data)
	}
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withDebugLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[debug] %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) handleBriefing(w http.ResponseWriter, r *http.Request) {
	briefing := s.hub.FetchAll()
	s.writeJSON(w, briefing)
}

func (s *Server) handleWeather(w http.ResponseWriter, r *http.Request) {
	weather := s.hub.Weather.GetCached()
	s.writeJSON(w, weather)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	events := s.hub.Calendar.GetCached()
	s.writeJSON(w, events)
}

func (s *Server) handleNews(w http.ResponseWriter, r *http.Request) {
	news := s.hub.News.GetCached()
	s.writeJSON(w, news)
}

func (s *Server) handleEmails(w http.ResponseWriter, r *http.Request) {
	emails := s.hub.Email.GetCached()
	s.writeJSON(w, emails)
}

func (s *Server) handleSlack(w http.ResponseWriter, r *http.Request) {
	msgs := s.hub.Slack.GetCached()
	s.writeJSON(w, msgs)
}

func (s *Server) handleGitHub(w http.ResponseWriter, r *http.Request) {
	notifs := s.hub.GitHub.GetCached()
	s.writeJSON(w, notifs)
}

func (s *Server) handleJira(w http.ResponseWriter, r *http.Request) {
	tickets := s.hub.Jira.GetCached()
	s.writeJSON(w, tickets)
}

func (s *Server) handleNotion(w http.ResponseWriter, r *http.Request) {
	pages := s.hub.Notion.GetCached()
	s.writeJSON(w, pages)
}

func (s *Server) handleTodos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		todos := s.hub.Todos.List()
		s.writeJSON(w, todos)
	case "POST":
		var item models.TodoItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		item.Source = "manual"
		item = s.hub.Todos.Add(item)
		s.wsHub.Broadcast(mustJSON(models.DashboardUpdate{Type: "todos_updated", Payload: s.hub.Todos.List()}))
		w.WriteHeader(http.StatusCreated)
		s.writeJSON(w, item)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTodoAction(w http.ResponseWriter, r *http.Request) {
	// Extract ID from /api/todos/{id} or /api/todos/{id}/complete
	rawPath := r.URL.Path[len("/api/todos/"):]
	if rawPath == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	// Handle /api/todos/{id}/complete
	if strings.HasSuffix(rawPath, "/complete") {
		id := strings.TrimSuffix(rawPath, "/complete")
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !s.hub.Todos.Complete(id) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		s.wsHub.Broadcast(mustJSON(models.DashboardUpdate{Type: "todos_updated", Payload: s.hub.Todos.List()}))
		// Return the updated todo
		for _, t := range s.hub.Todos.List() {
			if t.ID == id {
				s.writeJSON(w, t)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	id := rawPath
	switch r.Method {
	case "PUT", "PATCH":
		var updates struct {
			Title       *string               `json:"title"`
			Status      *models.TodoStatus    `json:"status"`
			Priority    *models.Priority      `json:"priority"`
			Notes       *string               `json:"notes"`
			Description *string               `json:"description"`
			Recurring   *models.RecurringRule `json:"recurring"`
		}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.hub.Todos.Update(id, func(t *models.TodoItem) {
			if updates.Title != nil {
				t.Title = *updates.Title
			}
			if updates.Status != nil {
				t.Status = *updates.Status
				if *updates.Status == models.TodoDone {
					now := time.Now()
					t.CompletedAt = &now
				}
			}
			if updates.Priority != nil {
				t.Priority = *updates.Priority
			}
			if updates.Notes != nil {
				t.Notes = *updates.Notes
			}
			if updates.Description != nil {
				t.Description = *updates.Description
			}
			if updates.Recurring != nil {
				t.Recurring = updates.Recurring
			}
		})
		s.wsHub.Broadcast(mustJSON(models.DashboardUpdate{Type: "todos_updated", Payload: s.hub.Todos.List()}))
		// Return the updated todo
		for _, t := range s.hub.Todos.List() {
			if t.ID == id {
				s.writeJSON(w, t)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	case "DELETE":
		s.hub.Todos.Delete(id)
		s.wsHub.Broadcast(mustJSON(models.DashboardUpdate{Type: "todos_updated", Payload: s.hub.Todos.List()}))
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSignals(w http.ResponseWriter, r *http.Request) {
	// Aggregate all signals into a unified feed
	var signals []models.Signal

	for _, msg := range s.hub.Slack.GetCached() {
		p := models.PriorityMedium
		if msg.IsUrgent {
			p = models.PriorityHigh
		}
		signals = append(signals, models.Signal{
			Source:    "slack",
			Type:      "message",
			Title:     fmt.Sprintf("%s in %s", msg.User, msg.Channel),
			Body:      msg.Text,
			Priority:  p,
			Timestamp: msg.Timestamp,
		})
	}

	for _, email := range s.hub.Email.GetCached() {
		p := models.PriorityMedium
		if email.IsStarred {
			p = models.PriorityHigh
		}
		signals = append(signals, models.Signal{
			Source:    "email",
			Type:      "email",
			Title:     email.Subject,
			Body:      email.Snippet,
			Priority:  p,
			Timestamp: email.Date,
			Read:      !email.IsUnread,
		})
	}

	for _, notif := range s.hub.GitHub.GetCached() {
		signals = append(signals, models.Signal{
			Source:    "github",
			Type:      notif.Type,
			Title:     notif.Title,
			Body:      fmt.Sprintf("%s — %s", notif.Repo, notif.Reason),
			URL:       notif.URL,
			Priority:  models.PriorityMedium,
			Timestamp: notif.UpdatedAt,
			Read:      !notif.Unread,
		})
	}

	s.writeJSON(w, signals)
}

// handleConfig serves GET/PUT for the server configuration.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.writeJSON(w, s.maskConfig(s.hub.GetConfig()))
	case "PUT":
		existing := s.hub.GetConfig()
		var incoming config.Config
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		merged := mergeConfig(existing, &incoming)
		if err := merged.Save(s.cfgPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Update in-memory config (non-credential fields take effect immediately)
		*existing = *merged
		s.writeJSON(w, map[string]interface{}{
			"ok":      true,
			"message": "Config saved. Restart the server for credential changes to take effect.",
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleReview generates an end-of-day review using the AI service.
func (s *Server) handleReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b := &models.Briefing{
		Date:          time.Now(),
		Todos:         s.hub.Todos.List(),
		Events:        s.hub.Calendar.GetCached(),
		UnreadEmails:  s.hub.Email.GetCached(),
		GitHubNotifs:  s.hub.GitHub.GetCached(),
		SlackMessages: s.hub.Slack.GetCached(),
	}
	review, err := s.hub.AI.DailyReview(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.writeJSON(w, map[string]string{"review": review})
}

// handleTimeBlocks generates focused-work time-block suggestions using the AI service.
func (s *Server) handleTimeBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b := &models.Briefing{
		Date:   time.Now(),
		Todos:  s.hub.Todos.List(),
		Events: s.hub.Calendar.GetCached(),
	}
	blocks, err := s.hub.AI.SuggestTimeBlocks(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if blocks == nil {
		blocks = []models.TimeBlock{}
	}
	s.writeJSON(w, map[string]interface{}{"blocks": blocks})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	websocket.ServeWs(s.wsHub, w, r)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/index.html")
}

// maskConfig returns a map representation of cfg with credential fields replaced
// by configRedacted when they are non-empty.
func (s *Server) maskConfig(cfg *config.Config) map[string]interface{} {
	data, err := json.Marshal(cfg)
	if err != nil {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]interface{}{}
	}

	redact := func(section string, fields ...string) {
		sec, ok := m[section].(map[string]interface{})
		if !ok {
			return
		}
		for _, f := range fields {
			if v, ok := sec[f].(string); ok && v != "" {
				sec[f] = configRedacted
			}
		}
	}

	redact("weather", "api_key")
	redact("news", "api_key")
	redact("google", "client_secret")
	redact("slack", "bot_token")
	redact("email", "password")
	redact("github", "token")
	redact("jira", "api_token")
	redact("notion", "token")
	redact("ai", "api_key")

	return m
}

// mergeConfig produces a new Config from incoming, preserving existing credential
// values wherever the incoming value is empty or configRedacted.
func mergeConfig(existing, incoming *config.Config) *config.Config {
	merged := *incoming

	keep := func(in, ex string) string {
		if in == configRedacted || in == "" {
			return ex
		}
		return in
	}

	merged.Weather.APIKey = keep(incoming.Weather.APIKey, existing.Weather.APIKey)
	merged.News.APIKey = keep(incoming.News.APIKey, existing.News.APIKey)
	merged.Google.ClientSecret = keep(incoming.Google.ClientSecret, existing.Google.ClientSecret)
	merged.Slack.BotToken = keep(incoming.Slack.BotToken, existing.Slack.BotToken)
	merged.Email.Password = keep(incoming.Email.Password, existing.Email.Password)
	merged.GitHub.Token = keep(incoming.GitHub.Token, existing.GitHub.Token)
	merged.Jira.Token = keep(incoming.Jira.Token, existing.Jira.Token)
	merged.Notion.Token = keep(incoming.Notion.Token, existing.Notion.Token)
	merged.AI.APIKey = keep(incoming.AI.APIKey, existing.AI.APIKey)

	return &merged
}

// ── Google OAuth handlers ─────────────────────────────────────────────────────

// handleAuthStatus returns the current connection status of all OAuth integrations.
func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.writeJSON(w, map[string]interface{}{
		"google": map[string]bool{
			"configured": s.hub.GoogleAuth.IsConfigured(),
			"connected":  s.hub.GoogleAuth.IsConnected(),
		},
	})
}

// handleAuthGoogle redirects the browser to Google's OAuth2 consent page.
func (s *Server) handleAuthGoogle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	returnTo := r.URL.Query().Get("return_to") // frontend origin for post-auth redirect
	authURL := s.hub.GoogleAuth.AuthURL(returnTo)
	if authURL == "" {
		http.Error(w, "Google OAuth not configured. Set google.client_id and google.client_secret in config.", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleAuthGoogleCallback receives the authorization code from Google and
// exchanges it for tokens, then redirects the browser back to the dashboard.
func (s *Server) handleAuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errParam := r.URL.Query().Get("error")

	if errParam != "" {
		http.Redirect(w, r, "/?google=denied", http.StatusFound)
		return
	}

	if !s.hub.GoogleAuth.ValidateState(state) {
		http.Error(w, "Invalid OAuth state — possible CSRF attack", http.StatusBadRequest)
		return
	}

	if err := s.hub.GoogleAuth.Exchange(r.Context(), code); err != nil {
		log.Printf("Google OAuth exchange error: %v", err)
		http.Redirect(w, r, "/?google=error", http.StatusFound)
		return
	}

	// Broadcast a refresh so the dashboard updates immediately.
	s.hub.Broadcast(models.DashboardUpdate{Type: "google_connected"})

	// Redirect back to the frontend (Next.js dev server or production origin).
	returnTo := s.hub.GoogleAuth.ReturnTo()
	if returnTo == "" {
		returnTo = "/" // fallback to Go's own dashboard
	}
	http.Redirect(w, r, returnTo+"?google=connected", http.StatusFound)
}

// handleAuthGoogleDisconnect deletes the stored Google token.
func (s *Server) handleAuthGoogleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.hub.GoogleAuth.Disconnect(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.writeJSON(w, map[string]string{"ok": "true"})
}

func mustJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return []byte("{}")
	}
	return data
}
