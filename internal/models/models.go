package models

import "time"

// Weather represents current weather data.
type Weather struct {
	City        string    `json:"city"`
	Temperature float64   `json:"temperature"`
	FeelsLike   float64   `json:"feels_like"`
	Humidity    int       `json:"humidity"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	WindSpeed   float64   `json:"wind_speed"`
	Units       string    `json:"units"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CalendarEvent represents a calendar event.
type CalendarEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	AllDay      bool      `json:"all_day"`
	MeetingURL  string    `json:"meeting_url"`
	Attendees   []string  `json:"attendees"`
	Source      string    `json:"source"`
}

// NewsItem represents a news article.
type NewsItem struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	ImageURL    string    `json:"image_url"`
}

// SlackMessage represents a Slack message.
type SlackMessage struct {
	Channel   string    `json:"channel"`
	User      string    `json:"user"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	ThreadTS  string    `json:"thread_ts"`
	IsUrgent  bool      `json:"is_urgent"`
	IsDM      bool      `json:"is_dm"`
}

// EmailMessage represents an email.
type EmailMessage struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	Subject   string    `json:"subject"`
	Snippet   string    `json:"snippet"`
	Date      time.Time `json:"date"`
	IsUnread  bool      `json:"is_unread"`
	IsStarred bool      `json:"is_starred"`
	Labels    []string  `json:"labels"`
	ThreadID  string    `json:"thread_id"`
}

// GitHubNotification represents a GitHub notification.
type GitHubNotification struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Repo      string    `json:"repo"`
	Type      string    `json:"type"`
	URL       string    `json:"url"`
	Reason    string    `json:"reason"`
	Unread    bool      `json:"unread"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JiraTicket represents a Jira issue.
type JiraTicket struct {
	Key      string    `json:"key"`
	Summary  string    `json:"summary"`
	Status   string    `json:"status"`
	Priority string    `json:"priority"`
	Assignee string    `json:"assignee"`
	DueDate  time.Time `json:"due_date"`
	URL      string    `json:"url"`
	Type     string    `json:"type"`
}

// NotionPage represents a page/task from a Notion database.
type NotionPage struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Status    string     `json:"status"`
	Priority  string     `json:"priority"`
	DueDate   *time.Time `json:"due_date,omitempty"`
	URL       string     `json:"url"`
	Database  string     `json:"database"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TodoItem represents a task in the interactive todo list.
type TodoItem struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    Priority   `json:"priority"`
	Status      TodoStatus `json:"status"`
	Source      string     `json:"source"`
	SourceID    string     `json:"source_id"`
	SourceURL   string     `json:"source_url"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Tags        []string   `json:"tags"`
	Notes       string     `json:"notes"`
}

type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityUrgent
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityUrgent:
		return "urgent"
	default:
		return "medium"
	}
}

type TodoStatus int

const (
	TodoPending TodoStatus = iota
	TodoInProgress
	TodoDone
	TodoArchived
)

func (s TodoStatus) String() string {
	switch s {
	case TodoPending:
		return "pending"
	case TodoInProgress:
		return "in_progress"
	case TodoDone:
		return "done"
	case TodoArchived:
		return "archived"
	default:
		return "pending"
	}
}

// Briefing is the aggregate morning briefing.
type Briefing struct {
	Date          time.Time            `json:"date"`
	Weather       *Weather             `json:"weather"`
	Events        []CalendarEvent      `json:"events"`
	News          []NewsItem           `json:"news"`
	UnreadEmails  []EmailMessage       `json:"unread_emails"`
	SlackMessages []SlackMessage       `json:"slack_messages"`
	GitHubNotifs  []GitHubNotification `json:"github_notifications"`
	JiraTickets   []JiraTicket         `json:"jira_tickets"`
	NotionPages   []NotionPage         `json:"notion_pages"`
	Todos         []TodoItem           `json:"todos"`
	EmailCount    int                  `json:"email_count"`
	SlackUnread   int                  `json:"slack_unread"`
	Summary       string               `json:"summary,omitempty"`
	GeneratedAt   time.Time            `json:"generated_at"`
}

// DashboardUpdate is sent over WebSocket for real-time updates.
type DashboardUpdate struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Signal represents an aggregated signal from any source.
type Signal struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	URL       string    `json:"url"`
	Priority  Priority  `json:"priority"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
}
