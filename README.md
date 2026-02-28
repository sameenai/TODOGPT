# Daily Briefing Dashboard

A full-stack productivity command center that pulls weather, calendar events, top news, Slack messages, emails, and GitHub notifications into your terminal and a real-time web dashboard — with an interactive todo list that helps you reach **inbox zero**.

## Quick Start

```bash
# Terminal briefing (prints to stdout)
./start.sh briefing

# Web dashboard (http://localhost:8080)
./start.sh

# Or using Make
make run
```

## Features

### Terminal Briefing (`./start.sh briefing`)
- Weather forecast with temperature, humidity, wind
- Today's calendar events with meeting links
- Top news headlines
- Unread email summary
- Slack message highlights (DMs, urgent messages)
- GitHub notifications (PRs, issues, review requests)
- Auto-generated action items from all sources

### Web Dashboard (`./start.sh`)
- **Real-time updates** via WebSocket — no page refresh needed
- **Score cards** showing unread counts across all channels
- **Inbox Zero Progress** — track how close you are to clearing all inboxes
- **Interactive Todo List** — auto-populated from emails, Slack, GitHub, calendar
  - Add/complete/delete tasks
  - Filter by status (all, pending, done, urgent)
  - Priority-sorted with source badges
- **Pomodoro Focus Timer** — 25/5 minute work/break cycles with notifications
- **Calendar view** with "happening now" highlighting and join links
- **Signal feed** from Slack, Email, GitHub in unified view

### Integrations

| Service | What It Provides | Config Key |
|---------|-----------------|------------|
| OpenWeatherMap | Weather data | `weather.api_key` |
| NewsAPI | Top headlines | `news.api_key` |
| Google Calendar | Today's events | `google.credentials_file` |
| Gmail | Unread emails | `google.credentials_file` |
| Slack | Channel messages, DMs | `slack.bot_token` |
| GitHub | PRs, issues, notifications | `github.token` |
| Jira | Assigned tickets | `jira.api_token` |
| Notion | Database items | `notion.token` |

## Configuration

Generate a default config:

```bash
./start.sh init
# Edit ~/.daily-briefing/config.json
```

Example config:
```json
{
  "server": { "port": 8080, "poll_interval_seconds": 30 },
  "weather": { "api_key": "YOUR_KEY", "city": "San Francisco", "enabled": true },
  "news": { "api_key": "YOUR_KEY", "max_items": 10, "enabled": true },
  "slack": { "bot_token": "xoxb-...", "channels": ["#general", "#engineering"], "enabled": true },
  "github": { "token": "ghp_...", "repos": ["myorg/myrepo"], "enabled": true }
}
```

The dashboard runs in **demo mode** with sample data when API keys are not configured, so you can explore the UI immediately.

## Architecture

```
cmd/
  briefing/     CLI terminal briefing
  server/       Web dashboard server
internal/
  api/          HTTP API + WebSocket bridge
  config/       Configuration management
  models/       Shared data models
  services/     Service integrations (weather, news, calendar, etc.)
  todo/         Persistent todo storage
  websocket/    WebSocket hub for real-time push
web/
  static/       CSS + JavaScript frontend
  templates/    HTML dashboard template
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/briefing` | Full briefing (all data) |
| GET | `/api/weather` | Weather data |
| GET | `/api/events` | Calendar events |
| GET | `/api/news` | News headlines |
| GET | `/api/emails` | Email messages |
| GET | `/api/slack` | Slack messages |
| GET | `/api/github` | GitHub notifications |
| GET | `/api/todos` | Todo list |
| POST | `/api/todos` | Add a todo |
| PATCH | `/api/todos/:id` | Update a todo |
| DELETE | `/api/todos/:id` | Delete a todo |
| GET | `/api/signals` | Unified signal feed |
| WS | `/ws` | WebSocket for real-time updates |
