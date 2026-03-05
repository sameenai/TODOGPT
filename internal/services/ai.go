package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

// claudeAPIURL is overridable in tests.
var claudeAPIURL = "https://api.anthropic.com/v1/messages"

// AIService generates a natural-language morning summary using Claude.
type AIService struct {
	cfg config.AIConfig
}

func NewAIService(cfg config.AIConfig) *AIService {
	return &AIService{cfg: cfg}
}

// apiKey returns the API key, preferring the env var over config.
func (s *AIService) apiKey() string {
	if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
		return k
	}
	return s.cfg.APIKey
}

// Summarize generates a concise morning briefing summary for the given Briefing.
// Returns an empty string (no error) when the service is disabled or unconfigured.
func (s *AIService) Summarize(b *models.Briefing) (string, error) {
	key := s.apiKey()
	if !s.cfg.Enabled || key == "" {
		return "", nil
	}

	model := s.cfg.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	prompt := buildPrompt(b)

	reqBody := map[string]interface{}{
		"model":      model,
		"max_tokens": 512,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, claudeAPIURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() // #nosec G307

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude API returned %d", resp.StatusCode)
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	for _, block := range result.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", nil
}

// buildPrompt constructs a concise briefing context for Claude.
func buildPrompt(b *models.Briefing) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "You are a personal assistant. Generate a concise, friendly morning briefing summary (3-5 sentences) based on the following data. Focus on the most important items that need attention today.\n\n")
	fmt.Fprintf(&buf, "Date: %s\n", b.Date.Format("Monday, January 2, 2006"))

	if b.Weather != nil {
		w := b.Weather
		fmt.Fprintf(&buf, "Weather: %.0f°F, %s, humidity %d%%, wind %.0f mph in %s\n",
			w.Temperature, w.Description, w.Humidity, w.WindSpeed, w.City)
	}

	if len(b.Events) > 0 {
		fmt.Fprintf(&buf, "Calendar: %d events today", len(b.Events))
		if len(b.Events) <= 3 {
			fmt.Fprintf(&buf, " (")
			for i, e := range b.Events {
				if i > 0 {
					fmt.Fprintf(&buf, ", ")
				}
				fmt.Fprintf(&buf, "%s at %s", e.Title, e.StartTime.Format("3:04 PM"))
			}
			fmt.Fprintf(&buf, ")")
		}
		fmt.Fprintln(&buf)
	}

	unreadEmails := 0
	for _, e := range b.UnreadEmails {
		if e.IsUnread {
			unreadEmails++
		}
	}
	if unreadEmails > 0 {
		fmt.Fprintf(&buf, "Email: %d unread messages\n", unreadEmails)
	}

	if len(b.SlackMessages) > 0 {
		urgent := 0
		for _, m := range b.SlackMessages {
			if m.IsUrgent || m.IsDM {
				urgent++
			}
		}
		fmt.Fprintf(&buf, "Slack: %d messages (%d urgent/DM)\n", len(b.SlackMessages), urgent)
	}

	ghUnread := 0
	for _, n := range b.GitHubNotifs {
		if n.Unread {
			ghUnread++
		}
	}
	if ghUnread > 0 {
		fmt.Fprintf(&buf, "GitHub: %d unread notifications\n", ghUnread)
	}

	pendingTodos := 0
	urgentTodos := 0
	for _, t := range b.Todos {
		if t.Status == models.TodoPending || t.Status == models.TodoInProgress {
			pendingTodos++
			if t.Priority == models.PriorityUrgent {
				urgentTodos++
			}
		}
	}
	if pendingTodos > 0 {
		fmt.Fprintf(&buf, "Tasks: %d pending (%d urgent)\n", pendingTodos, urgentTodos)
		for i, t := range b.Todos {
			if i >= 3 {
				break
			}
			if t.Status == models.TodoPending || t.Status == models.TodoInProgress {
				fmt.Fprintf(&buf, "  - [%s] %s\n", t.Priority.String(), t.Title)
			}
		}
	}

	if len(b.JiraTickets) > 0 {
		fmt.Fprintf(&buf, "Jira: %d open tickets\n", len(b.JiraTickets))
	}

	fmt.Fprintf(&buf, "\nWrite the summary now:")
	return buf.String()
}
