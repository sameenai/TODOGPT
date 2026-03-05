package services

import (
	"log"
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
	"github.com/todogpt/daily-briefing/internal/todo"
)

// Hub aggregates all services and manages polling/updates.
type Hub struct {
	Weather  *WeatherService
	News     *NewsService
	Calendar *CalendarService
	Slack    *SlackService
	Email    *EmailService
	GitHub   *GitHubService
	Jira     *JiraService
	Notion   *NotionService
	AI       *AIService
	Todos    *TodoService

	cfg       *config.Config
	listeners []chan models.DashboardUpdate
	mu        sync.RWMutex
	stopCh    chan struct{}
}

func NewHub(cfg *config.Config) *Hub {
	todosvc := newTodoService(cfg)
	return &Hub{
		Weather:  NewWeatherService(cfg.Weather),
		News:     NewNewsService(cfg.News),
		Calendar: NewCalendarService(cfg.Google),
		Slack:    NewSlackService(cfg.Slack),
		Email:    NewEmailService(cfg.Email),
		GitHub:   NewGitHubService(cfg.GitHub),
		Jira:     NewJiraService(cfg.Jira),
		Notion:   NewNotionService(cfg.Notion),
		AI:       NewAIService(cfg.AI),
		Todos:    todosvc,
		cfg:      cfg,
		stopCh:   make(chan struct{}),
	}
}

// newTodoService creates a TodoService, attaching a persistent Store when
// possible. Falls back to an in-memory-only service on error.
func newTodoService(cfg *config.Config) *TodoService {
	// When DataDir is set (e.g. in tests), use an explicit path instead of the
	// default ~/.daily-briefing directory.
	storePath := ""
	if cfg.Server.DataDir != "" {
		storePath = cfg.Server.DataDir + "/todos.json"
	}
	store, err := todo.NewStore(storePath)
	if err != nil {
		log.Printf("todo store unavailable, using in-memory todos: %v", err)
		return NewTodoService()
	}
	return NewTodoServiceWithStore(store)
}

func (h *Hub) Subscribe() chan models.DashboardUpdate {
	ch := make(chan models.DashboardUpdate, 50)
	h.mu.Lock()
	h.listeners = append(h.listeners, ch)
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan models.DashboardUpdate) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, l := range h.listeners {
		if l == ch {
			h.listeners = append(h.listeners[:i], h.listeners[i+1:]...)
			close(ch)
			return
		}
	}
}

func (h *Hub) Broadcast(update models.DashboardUpdate) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.listeners {
		select {
		case ch <- update:
		default:
			// Drop if listener is slow
		}
	}
}

func (h *Hub) FetchAll() *models.Briefing {
	briefing := &models.Briefing{
		Date:        time.Now(),
		GeneratedAt: time.Now(),
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		w, err := h.Weather.Fetch()
		if err != nil {
			log.Printf("Weather fetch error: %v", err)
			return
		}
		briefing.Weather = w
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		n, err := h.News.Fetch()
		if err != nil {
			log.Printf("News fetch error: %v", err)
			return
		}
		briefing.News = n
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		e, err := h.Calendar.Fetch()
		if err != nil {
			log.Printf("Calendar fetch error: %v", err)
			return
		}
		briefing.Events = e
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		msgs, err := h.Slack.Fetch()
		if err != nil {
			log.Printf("Slack fetch error: %v", err)
			return
		}
		briefing.SlackMessages = msgs
		count := 0
		for _, m := range msgs {
			if m.IsUrgent || m.IsDM {
				count++
			}
		}
		briefing.SlackUnread = count
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		emails, err := h.Email.Fetch()
		if err != nil {
			log.Printf("Email fetch error: %v", err)
			return
		}
		briefing.UnreadEmails = emails
		briefing.EmailCount = h.Email.UnreadCount()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		notifs, err := h.GitHub.Fetch()
		if err != nil {
			log.Printf("GitHub fetch error: %v", err)
			return
		}
		briefing.GitHubNotifs = notifs
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tickets, err := h.Jira.Fetch()
		if err != nil {
			log.Printf("Jira fetch error: %v", err)
			return
		}
		briefing.JiraTickets = tickets
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		pages, err := h.Notion.Fetch()
		if err != nil {
			log.Printf("Notion fetch error: %v", err)
			return
		}
		briefing.NotionPages = pages
	}()

	wg.Wait()

	// Auto-generate todos from signals
	h.Todos.GenerateFromBriefing(briefing)
	briefing.Todos = h.Todos.List()

	// Generate AI summary (non-blocking — failures are silently ignored)
	if summary, err := h.AI.Summarize(briefing); err == nil && summary != "" {
		briefing.Summary = summary
	} else if err != nil {
		log.Printf("AI summary error: %v", err)
	}

	return briefing
}

func (h *Hub) StartPolling() {
	interval := time.Duration(h.cfg.Server.PollInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial fetch
	briefing := h.FetchAll()
	h.Broadcast(models.DashboardUpdate{Type: "full_refresh", Payload: briefing})

	for {
		select {
		case <-ticker.C:
			briefing := h.FetchAll()
			h.Broadcast(models.DashboardUpdate{Type: "full_refresh", Payload: briefing})
		case <-h.stopCh:
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.stopCh)
}
