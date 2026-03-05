package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

// claudeAPIURL is overridable in tests.
var claudeAPIURL = "https://api.anthropic.com/v1/messages"

// ExportClaudeAPIURL returns the current claudeAPIURL for test helpers outside this package.
func ExportClaudeAPIURL() string { return claudeAPIURL }

// SetClaudeAPIURL sets the claudeAPIURL for test helpers outside this package.
func SetClaudeAPIURL(url string) { claudeAPIURL = url }

// AIService generates natural-language content using Claude.
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

// callClaude sends a prompt to the Claude API and returns the text response.
// Returns ("", nil) when the service is disabled or unconfigured.
func (s *AIService) callClaude(prompt string, maxTokens int) (string, error) {
	key := s.apiKey()
	if !s.cfg.Enabled || key == "" {
		return "", nil
	}

	model := s.cfg.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	reqBody := map[string]interface{}{
		"model":      model,
		"max_tokens": maxTokens,
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

// Summarize generates a concise morning briefing summary for the given Briefing.
// Returns an empty string (no error) when the service is disabled or unconfigured.
func (s *AIService) Summarize(b *models.Briefing) (string, error) {
	return s.callClaude(buildPrompt(b), 512)
}

// DailyReview generates an end-of-day review summarising what was accomplished
// and what to carry forward. Returns ("", nil) when AI is disabled.
func (s *AIService) DailyReview(b *models.Briefing) (string, error) {
	return s.callClaude(buildReviewPrompt(b), 1024)
}

// SuggestTimeBlocks analyses the calendar and pending todos and returns
// suggested focused-work blocks. Returns (nil, nil) when AI is disabled.
func (s *AIService) SuggestTimeBlocks(b *models.Briefing) ([]models.TimeBlock, error) {
	text, err := s.callClaude(buildTimeBlockPrompt(b), 1024)
	if err != nil || text == "" {
		return nil, err
	}
	return parseTimeBlocks(text)
}

// parseTimeBlocks extracts a JSON array of TimeBlock from a Claude response
// that may be wrapped in markdown code fences.
func parseTimeBlocks(text string) ([]models.TimeBlock, error) {
	text = strings.TrimSpace(text)
	// Strip optional markdown code fence
	if idx := strings.Index(text, "```"); idx >= 0 {
		text = text[idx:]
		if end := strings.Index(text[3:], "```"); end >= 0 {
			text = text[3 : 3+end]
		}
		// Strip language tag (e.g. "json\n")
		if nl := strings.Index(text, "\n"); nl >= 0 {
			text = text[nl+1:]
		}
	}
	// Find the JSON array boundaries
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end < start {
		return nil, fmt.Errorf("no JSON array found in time block response")
	}
	text = text[start : end+1]

	var blocks []models.TimeBlock
	if err := json.Unmarshal([]byte(text), &blocks); err != nil {
		return nil, fmt.Errorf("parse time blocks: %w", err)
	}
	return blocks, nil
}

// ── Prompt builders ──────────────────────────────────────────────────────────

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

// buildReviewPrompt builds an end-of-day review prompt.
func buildReviewPrompt(b *models.Briefing) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "You are a productivity coach. Generate a brief, encouraging end-of-day review.\n\n")
	fmt.Fprintf(&buf, "Date: %s\n\n", b.Date.Format("Monday, January 2, 2006"))

	completed := []models.TodoItem{}
	pending := []models.TodoItem{}
	for _, t := range b.Todos {
		if t.Status == models.TodoDone {
			completed = append(completed, t)
		} else if t.Status == models.TodoPending || t.Status == models.TodoInProgress {
			pending = append(pending, t)
		}
	}

	fmt.Fprintf(&buf, "COMPLETED TODAY (%d tasks):\n", len(completed))
	for i, t := range completed {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&buf, "- %s\n", t.Title)
	}

	fmt.Fprintf(&buf, "\nSTILL PENDING (%d tasks):\n", len(pending))
	for i, t := range pending {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&buf, "- [%s] %s\n", t.Priority.String(), t.Title)
	}

	if len(b.Events) > 0 {
		fmt.Fprintf(&buf, "\nEVENTS TODAY (%d):\n", len(b.Events))
		for i, e := range b.Events {
			if i >= 3 {
				break
			}
			fmt.Fprintf(&buf, "- %s\n", e.Title)
		}
	}

	fmt.Fprintf(&buf, "\nWrite a short review (3-5 sentences) covering: what was accomplished, what to carry forward tomorrow, and one improvement tip. Be encouraging and specific.")
	return buf.String()
}

// buildTimeBlockPrompt builds a prompt for time-block suggestions.
func buildTimeBlockPrompt(b *models.Briefing) string {
	var buf bytes.Buffer

	now := time.Now()
	fmt.Fprintf(&buf, "You are a productivity assistant. Today is %s.\n\n", now.Format("Monday, January 2, 2006"))
	fmt.Fprintf(&buf, "Suggest time blocks for focused work. Respond with ONLY a JSON array. No other text.\n\n")

	if len(b.Events) > 0 {
		fmt.Fprintf(&buf, "SCHEDULED EVENTS (already blocked):\n")
		for _, e := range b.Events {
			if e.AllDay {
				fmt.Fprintf(&buf, "- All day: %s\n", e.Title)
			} else {
				fmt.Fprintf(&buf, "- %s–%s: %s\n",
					e.StartTime.Format("15:04"),
					e.EndTime.Format("15:04"),
					e.Title)
			}
		}
	} else {
		fmt.Fprintf(&buf, "SCHEDULED EVENTS: none\n")
	}

	pending := []models.TodoItem{}
	for _, t := range b.Todos {
		if t.Status == models.TodoPending || t.Status == models.TodoInProgress {
			pending = append(pending, t)
		}
	}

	if len(pending) > 0 {
		fmt.Fprintf(&buf, "\nPENDING TASKS:\n")
		for _, t := range pending {
			fmt.Fprintf(&buf, "- [%s] id:%s — %s\n", t.Priority.String(), t.ID, t.Title)
		}
	}

	fmt.Fprintf(&buf, `
Return a JSON array. Each element must have exactly these fields:
{"start":"HH:MM","end":"HH:MM","title":"...","todo_id":"...","notes":"...","color":"..."}
- start/end: 24-hour format (e.g. "09:00")
- todo_id: ID of the main task (omit if not tied to one task)
- notes: brief rationale (optional, omit if none)
- color: "red" (urgent), "orange" (high), "blue" (medium), "gray" (low/admin)
Suggest 3-6 blocks in the 09:00-18:00 window. Avoid calendar event times.`)

	return buf.String()
}
