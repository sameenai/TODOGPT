package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/todogpt/daily-briefing/internal/models"
	"github.com/todogpt/daily-briefing/internal/services"
	"github.com/todogpt/daily-briefing/internal/websocket"
)

type Server struct {
	hub    *services.Hub
	wsHub  *websocket.Hub
	mux    *http.ServeMux
	port   int
	host   string
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

func (s *Server) registerRoutes() {
	// API routes
	s.mux.HandleFunc("/api/briefing", s.handleBriefing)
	s.mux.HandleFunc("/api/weather", s.handleWeather)
	s.mux.HandleFunc("/api/events", s.handleEvents)
	s.mux.HandleFunc("/api/news", s.handleNews)
	s.mux.HandleFunc("/api/emails", s.handleEmails)
	s.mux.HandleFunc("/api/slack", s.handleSlack)
	s.mux.HandleFunc("/api/github", s.handleGitHub)
	s.mux.HandleFunc("/api/todos", s.handleTodos)
	s.mux.HandleFunc("/api/todos/", s.handleTodoAction)
	s.mux.HandleFunc("/api/signals", s.handleSignals)

	// WebSocket
	s.mux.HandleFunc("/ws", s.handleWebSocket)

	// Static files
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Dashboard
	s.mux.HandleFunc("/", s.handleDashboard)
}

func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	// Bridge service hub updates to WebSocket clients
	go s.bridgeUpdates()

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	log.Printf("Dashboard running at http://%s", addr)
	return http.ListenAndServe(addr, s.withCORS(s.mux))
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

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
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
		s.hub.Todos.Add(item)
		s.wsHub.Broadcast(mustJSON(models.DashboardUpdate{Type: "todos_updated", Payload: s.hub.Todos.List()}))
		s.writeJSON(w, item)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTodoAction(w http.ResponseWriter, r *http.Request) {
	// Extract ID from /api/todos/{id}
	id := r.URL.Path[len("/api/todos/"):]
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "PUT", "PATCH":
		var updates struct {
			Title       *string          `json:"title"`
			Status      *models.TodoStatus `json:"status"`
			Priority    *models.Priority   `json:"priority"`
			Notes       *string          `json:"notes"`
			Description *string          `json:"description"`
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
		})
		s.wsHub.Broadcast(mustJSON(models.DashboardUpdate{Type: "todos_updated", Payload: s.hub.Todos.List()}))
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

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	websocket.ServeWs(s.wsHub, w, r)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/index.html")
}

func mustJSON(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
