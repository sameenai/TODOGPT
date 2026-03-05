package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/todogpt/daily-briefing/internal/models"
)

func (m model) renderNews() string {
	news := m.briefing.News
	if len(news) == 0 {
		return "  No news available."
	}
	limit := 10
	if len(news) < limit {
		limit = len(news)
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  Top %d Stories", limit)) + "\n\n")
	for i, n := range news {
		if i >= limit {
			break
		}
		sb.WriteString(fmt.Sprintf("%s  %s\n",
			dimStyle.Render(fmt.Sprintf("  %2d.", i+1)), n.Title))
		sb.WriteString(fmt.Sprintf("       %s\n\n",
			dimStyle.Render(n.Source+" · "+n.PublishedAt.Format("Jan 2, 3:04 PM"))))
	}
	return sb.String()
}

func (m model) renderWeather() string {
	w := m.briefing.Weather
	if w == nil {
		return "  No weather data."
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  %s", w.City)) + "\n\n")
	sb.WriteString(fmt.Sprintf("  %.0f°F  (feels like %.0f°F)\n", w.Temperature, w.FeelsLike))
	sb.WriteString(fmt.Sprintf("  %s\n", w.Description))
	sb.WriteString(fmt.Sprintf("  Humidity: %d%%   Wind: %.0f mph\n", w.Humidity, w.WindSpeed))
	return sb.String()
}

func (m model) renderCalendar() string {
	events := m.briefing.Events
	if len(events) == 0 {
		return "  No events today."
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  %d events today", len(events))) + "\n\n")
	for _, e := range events {
		timeStr := e.StartTime.Format("3:04") + " – " + e.EndTime.Format("3:04 PM")
		loc := ""
		if e.Location != "" {
			loc = "  @ " + e.Location
		} else if e.MeetingURL != "" {
			loc = "  (virtual)"
		}
		sb.WriteString(fmt.Sprintf("  %s   %s%s\n",
			dimStyle.Render(timeStr), e.Title, dimStyle.Render(loc)))
	}
	return sb.String()
}

func (m model) renderEmail() string {
	unread := 0
	for _, e := range m.briefing.UnreadEmails {
		if e.IsUnread {
			unread++
		}
	}
	if unread == 0 {
		return successStyle.Render("  Inbox zero! No unread emails.") + "\n"
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  %d unread emails", unread)) + "\n\n")
	for _, e := range m.briefing.UnreadEmails {
		if !e.IsUnread {
			continue
		}
		marker := " "
		if e.IsStarred {
			marker = "★"
		}
		maxSubj := m.width - 8
		if maxSubj < 10 {
			maxSubj = 10
		}
		sb.WriteString(fmt.Sprintf("  %s  %s\n", marker, truncate(e.Subject, maxSubj)))
		sb.WriteString(fmt.Sprintf("      %s\n\n", dimStyle.Render(e.From+" · "+e.Date.Format("Jan 2"))))
	}
	return sb.String()
}

func (m model) renderSlack() string {
	msgs := m.briefing.SlackMessages
	if len(msgs) == 0 {
		return "  No Slack messages."
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  %d messages", len(msgs))) + "\n\n")
	for _, msg := range msgs {
		marker := " "
		if msg.IsDM {
			marker = "@"
		} else if msg.IsUrgent {
			marker = "!"
		}
		maxText := m.width - 22
		if maxText < 10 {
			maxText = 10
		}
		ch := lipgloss.NewStyle().Width(14).Render(msg.Channel)
		sb.WriteString(fmt.Sprintf("  %s  %s  %s\n", marker, ch, dimStyle.Render(msg.User)))
		sb.WriteString(fmt.Sprintf("       %s\n\n", truncate(msg.Text, maxText)))
	}
	return sb.String()
}

func (m model) renderGitHub() string {
	unread := 0
	for _, n := range m.briefing.GitHubNotifs {
		if n.Unread {
			unread++
		}
	}
	if unread == 0 {
		return successStyle.Render("  No unread GitHub notifications.") + "\n"
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  %d notifications", unread)) + "\n\n")
	for _, n := range m.briefing.GitHubNotifs {
		if !n.Unread {
			continue
		}
		typeStr := "PR"
		if n.Type == "Issue" {
			typeStr = "IS"
		}
		maxTitle := m.width - 14
		if maxTitle < 10 {
			maxTitle = 10
		}
		sb.WriteString(fmt.Sprintf("  [%s]  %s\n", typeStr, truncate(n.Title, maxTitle)))
		sb.WriteString(fmt.Sprintf("        %s\n\n", dimStyle.Render(n.Repo+" · "+n.Reason)))
	}
	return sb.String()
}

func (m model) renderTodos() string {
	pending := filterPending(m.briefing.Todos)
	if len(pending) == 0 {
		return successStyle.Render("  All done! No pending action items.") + "\n"
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  %d action items", len(pending))) + "\n\n")
	for i, t := range pending {
		var priorityStr string
		switch t.Priority {
		case models.PriorityUrgent:
			priorityStr = urgentStyle.Render("[urgent]")
		case models.PriorityHigh:
			priorityStr = highStyle.Render("[high]  ")
		case models.PriorityMedium:
			priorityStr = dimStyle.Render("[med]   ")
		default:
			priorityStr = dimStyle.Render("[low]   ")
		}

		cursor := "  "
		title := t.Title
		if i == m.selectedTodo {
			cursor = cursorStyle.Render("▸ ")
			title = selectedStyle.Render(t.Title)
		}
		sb.WriteString(fmt.Sprintf("  %s%s  %s\n", cursor, priorityStr, title))
		sb.WriteString(fmt.Sprintf("             %s\n\n", dimStyle.Render(t.Source)))
	}
	sb.WriteString(dimStyle.Render("  space/enter: mark done  ↑/↓: navigate") + "\n")
	return sb.String()
}
