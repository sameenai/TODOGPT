# Daily Briefing Dashboard

A full-stack productivity command center that pulls weather, calendar events, top news, Slack messages, emails, and GitHub notifications into your terminal and a real-time web dashboard — with an interactive todo list that helps you reach **inbox zero**.

## Prerequisites

- **Go 1.21+** — [Install Go](https://go.dev/doc/install)
- **Git** — to clone the repo

That's it. No Node.js, no Docker, no external databases.

## Quick Start

```bash
# 1. Clone and enter the project
git clone https://github.com/sameenai/TODOGPT.git
cd TODOGPT

# 2. Build and start the web dashboard
make run
```

Open **http://localhost:8080** in your browser. The dashboard starts immediately in demo mode with sample data — no API keys needed to explore.

### Other ways to run

```bash
# Print a terminal briefing to stdout (no browser needed)
make briefing

# Start on a different port
./start.sh --port 9090

# Generate a config file to add your API keys
make init
# Then edit ~/.daily-briefing/config.json
```

## What You Get

### Terminal Briefing

Run `make briefing` to get a colorized morning summary printed to your terminal:

- Weather (temperature, humidity, wind)
- Today's calendar events with meeting links
- Top news headlines
- Unread email summary
- Slack highlights (DMs, urgent messages)
- GitHub notifications (PRs, issues, review requests)
- Auto-generated action items from all sources

### Web Dashboard

Run `make run` to start the live dashboard at http://localhost:8080:

- **Real-time updates** via WebSocket — no page refresh needed
- **Score cards** showing unread counts across all channels
- **Inbox Zero Progress** — track how close you are to clearing every inbox
- **Interactive Todo List** — auto-populated from emails, Slack, GitHub, calendar
  - Add, complete, and delete tasks
  - Filter by status (all, pending, done, urgent)
  - Priority-sorted with source badges
- **Pomodoro Focus Timer** — 25/5 minute work/break cycles with browser notifications
- **Calendar view** with "happening now" highlighting and meeting join links
- **Signal feed** from Slack, Email, and GitHub in one unified stream

## Connecting Your Accounts

The dashboard works out of the box with demo data. To connect real services, generate a config and add your API keys:

```bash
make init
# Creates ~/.daily-briefing/config.json
```

Then edit the file:

```json
{
  "server": {
    "port": 8080,
    "poll_interval_seconds": 30
  },
  "weather": {
    "api_key": "YOUR_OPENWEATHERMAP_KEY",
    "city": "San Francisco",
    "enabled": true
  },
  "news": {
    "api_key": "YOUR_NEWSAPI_KEY",
    "max_items": 10,
    "enabled": true
  },
  "slack": {
    "bot_token": "xoxb-...",
    "channels": ["#general", "#engineering"],
    "enabled": true
  },
  "github": {
    "token": "ghp_...",
    "repos": ["myorg/myrepo"],
    "enabled": true
  }
}
```

### Where to get API keys

| Service | Get a key | Config field |
|---------|-----------|--------------|
| OpenWeatherMap | https://openweathermap.org/api | `weather.api_key` |
| NewsAPI | https://newsapi.org/register | `news.api_key` |
| Google Calendar/Gmail | Google Cloud Console (OAuth2) | `google.credentials_file` |
| Slack | https://api.slack.com/apps | `slack.bot_token` |
| GitHub | Settings > Developer settings > Tokens | `github.token` |
| Jira | Atlassian account > API tokens | `jira.api_token` |
| Notion | https://www.notion.so/my-integrations | `notion.token` |

## Development

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report
make test-coverage

# Run linter + tests + build (same as CI)
make ci

# Clean build artifacts
make clean
```

## Project Structure

```
cmd/
  briefing/        CLI terminal briefing
  server/          Web dashboard server
internal/
  api/             HTTP handlers + WebSocket bridge
  config/          Configuration management
  models/          Shared data models
  services/        Service integrations (weather, news, calendar, etc.)
  todo/            Persistent todo storage
  websocket/       WebSocket hub for real-time push
web/
  static/css/      Dashboard styles
  static/js/       Dashboard JavaScript (vanilla, no framework)
  templates/       HTML template
```

## API

All endpoints return JSON. The dashboard uses these internally, but you can also call them directly.

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
