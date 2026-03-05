package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
	"github.com/todogpt/daily-briefing/internal/services"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("")
	if err != nil {
		return err
	}

	hub := services.NewHub(cfg)
	briefing := hub.FetchAll()

	printAll(briefing)
	return nil
}

func printAll(b *models.Briefing) {
	printHeaderAt(time.Now())
	printSummary(b)
	printWeather(b)
	printCalendar(b)
	printNews(b)
	printEmails(b)
	printSlack(b)
	printGitHub(b)
	printTodos(b)
	printFooter(b)
}

func printHeaderAt(now time.Time) {
	greeting := "Good morning"
	hour := now.Hour()
	if hour >= 12 && hour < 17 {
		greeting = "Good afternoon"
	} else if hour >= 17 {
		greeting = "Good evening"
	}

	fmt.Println()
	fmt.Printf("%s%s%s\n", colorBold, strings.Repeat("=", 60), colorReset)
	fmt.Printf("  %s%s%s! Today is %s\n", colorBold, greeting, colorReset, now.Format("Monday, January 2, 2006"))
	fmt.Printf("%s%s%s\n\n", colorBold, strings.Repeat("=", 60), colorReset)
}

func printSummary(b *models.Briefing) {
	if b.Summary == "" {
		return
	}
	fmt.Printf("%s  AI BRIEFING SUMMARY%s\n", colorBold+colorCyan, colorReset)
	fmt.Printf("%s  %s%s\n\n", colorBold, b.Summary, colorReset)
}

func printWeather(b *models.Briefing) {
	if b.Weather == nil {
		return
	}
	w := b.Weather
	fmt.Printf("%s  WEATHER — %s%s\n", colorCyan, w.City, colorReset)
	fmt.Printf("  %.0f°F (feels like %.0f°F) | %s | Humidity: %d%% | Wind: %.0f mph\n\n",
		w.Temperature, w.FeelsLike, w.Description, w.Humidity, w.WindSpeed)
}

func printCalendar(b *models.Briefing) {
	if len(b.Events) == 0 {
		return
	}
	fmt.Printf("%s  CALENDAR — %d events today%s\n", colorBlue, len(b.Events), colorReset)
	for _, e := range b.Events {
		timeStr := e.StartTime.Format("3:04 PM")
		endStr := e.EndTime.Format("3:04 PM")
		loc := ""
		if e.Location != "" {
			loc = fmt.Sprintf(" @ %s", e.Location)
		} else if e.MeetingURL != "" {
			loc = " (virtual)"
		}
		fmt.Printf("  %s%-10s%s %s — %s%s\n", colorDim, timeStr+" - "+endStr, colorReset, e.Title, loc, "")
	}
	fmt.Println()
}

func printNews(b *models.Briefing) {
	if len(b.News) == 0 {
		return
	}
	fmt.Printf("%s  TOP NEWS%s\n", colorGreen, colorReset)
	for i, n := range b.News {
		if i >= 5 {
			break
		}
		fmt.Printf("  %s%d.%s %s %s(%s)%s\n", colorBold, i+1, colorReset, n.Title, colorDim, n.Source, colorReset)
	}
	fmt.Println()
}

func printEmails(b *models.Briefing) {
	if len(b.UnreadEmails) == 0 {
		return
	}
	unread := 0
	for _, e := range b.UnreadEmails {
		if e.IsUnread {
			unread++
		}
	}
	fmt.Printf("%s  EMAIL — %d unread%s\n", colorYellow, unread, colorReset)
	for _, e := range b.UnreadEmails {
		if !e.IsUnread {
			continue
		}
		marker := " "
		if e.IsStarred {
			marker = "*"
		}
		fmt.Printf("  %s %s%-30s%s %s\n", marker, colorBold, truncate(e.Subject, 30), colorReset, colorDim+e.From+colorReset)
	}
	fmt.Println()
}

func printSlack(b *models.Briefing) {
	if len(b.SlackMessages) == 0 {
		return
	}
	fmt.Printf("%s  SLACK — %d messages%s\n", colorRed, len(b.SlackMessages), colorReset)
	for _, m := range b.SlackMessages {
		urgentMark := " "
		if m.IsUrgent {
			urgentMark = "!"
		}
		if m.IsDM {
			urgentMark = "@"
		}
		fmt.Printf("  %s %-15s %s%s%s %s\n",
			urgentMark,
			m.Channel,
			colorDim, m.User, colorReset,
			truncate(m.Text, 50))
	}
	fmt.Println()
}

func printGitHub(b *models.Briefing) {
	unread := 0
	for _, n := range b.GitHubNotifs {
		if n.Unread {
			unread++
		}
	}
	if unread == 0 {
		return
	}
	fmt.Printf("%s  GITHUB — %d notifications%s\n", colorCyan, unread, colorReset)
	for _, n := range b.GitHubNotifs {
		if !n.Unread {
			continue
		}
		typeIcon := "PR"
		if n.Type == "Issue" {
			typeIcon = "IS"
		}
		fmt.Printf("  [%s] %-40s %s%s — %s%s\n",
			typeIcon, truncate(n.Title, 40), colorDim, n.Repo, n.Reason, colorReset)
	}
	fmt.Println()
}

func printTodos(b *models.Briefing) {
	pending := 0
	for _, t := range b.Todos {
		if t.Status == models.TodoPending || t.Status == models.TodoInProgress {
			pending++
		}
	}
	if pending == 0 {
		return
	}
	fmt.Printf("%s  ACTION ITEMS — %d tasks%s\n", colorRed+colorBold, pending, colorReset)
	for i, t := range b.Todos {
		if t.Status == models.TodoDone || t.Status == models.TodoArchived {
			continue
		}
		priorityColor := colorReset
		switch t.Priority {
		case models.PriorityUrgent:
			priorityColor = colorRed
		case models.PriorityHigh:
			priorityColor = colorYellow
		}
		fmt.Printf("  %s%d. [%s] %s%s %s(%s)%s\n",
			priorityColor, i+1, t.Priority.String(), t.Title, colorReset,
			colorDim, t.Source, colorReset)
	}
	fmt.Println()
}

func printFooter(b *models.Briefing) {
	fmt.Printf("%s%s%s\n", colorDim, strings.Repeat("-", 60), colorReset)
	fmt.Printf("  %sDashboard: http://localhost:8080%s\n", colorCyan, colorReset)
	fmt.Printf("  %sGenerated at %s%s\n", colorDim, b.GeneratedAt.Format("3:04:05 PM"), colorReset)
	fmt.Printf("%s%s%s\n\n", colorDim, strings.Repeat("-", 60), colorReset)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
