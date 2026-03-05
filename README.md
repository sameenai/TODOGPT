# Daily Briefing Dashboard

A productivity command center that aggregates weather, news, calendar events, GitHub notifications, Jira tickets, and Notion pages into a real-time web dashboard — with an AI-powered morning summary, an interactive todo list, and a Pomodoro focus timer.

## What works out of the box

When you first run the app, two integrations are **live immediately** — no API keys required:

| Integration | Data source | Status |
|-------------|-------------|--------|
| Weather | [Open-Meteo](https://open-meteo.com/) (free, no key) | **LIVE** |
| News | [Hacker News](https://news.ycombinator.com/) (free, no key) | **LIVE** |
| Calendar | iCal URL from Google Calendar / iCloud / Outlook | DEMO until configured |
| GitHub | Personal access token | DEMO until configured |
| Jira | Atlassian API token | DEMO until configured |
| Notion | Integration token | DEMO until configured |
| Email | Not yet implemented | — |
| Slack | Not yet implemented | — |

Sections that aren't configured show a **Connect** prompt with step-by-step instructions inline — no fake data is shown.

## Prerequisites

- **Go 1.21+** — [Install Go](https://go.dev/doc/install)
- **Node.js 18+** — [Install Node.js](https://nodejs.org/) (for the web dashboard)

## Running locally

The app has two parts: a Go API server and a Next.js frontend. Run them in separate terminals.

**Terminal 1 — Go backend:**
```bash
make run
# Starts the API server at http://localhost:8080
```

**Terminal 2 — Next.js frontend:**
```bash
make frontend-install   # first time only
make frontend-dev
# Starts the dashboard at http://localhost:3000
```

Open **http://localhost:3000** in your browser.

The backend also serves a plain HTML dashboard at **http://localhost:8080** if you prefer not to run Node.js.

## Configuring integrations

Generate a config file:
```bash
make init
# Creates ~/.daily-briefing/config.json
```

Then follow the steps below for each integration you want to connect. After editing the config, restart the Go backend (`make run`).

### Calendar (Google Calendar, iCloud, Outlook)

Calendar works with any service that provides a private iCal subscription URL — no OAuth required.

**Google Calendar:**
1. Open [Google Calendar](https://calendar.google.com) → Settings (⚙) → select a calendar from the left sidebar
2. Scroll to **Integrate calendar** → copy the **Secret address in iCal format** URL

**iCloud:**
1. Open Calendar on Mac → right-click a calendar → **Share Calendar** → enable **Public Calendar** → copy the link
2. Replace `webcal://` with `https://`

**Outlook / Microsoft 365:**
1. Open [Outlook Web](https://outlook.live.com) → Settings → Calendar → Shared calendars → Publish a calendar
2. Copy the ICS link

Add to `~/.daily-briefing/config.json`:
```json
"google": {
  "ical_url": "https://calendar.google.com/calendar/ical/.../basic.ics"
}
```

### GitHub

1. Go to [GitHub Settings → Developer settings → Personal access tokens](https://github.com/settings/tokens)
2. Generate a **classic token** with the `notifications` scope
3. Add to config:
```json
"github": {
  "enabled": true,
  "token": "ghp_your_token_here"
}
```

### Jira

1. Go to [Atlassian account settings](https://id.atlassian.com/manage-profile/security/api-tokens) → **Create API token**
2. Find your Jira base URL (e.g. `https://yourcompany.atlassian.net`)
3. Find your project key (e.g. `ENG`)
4. Add to config:
```json
"jira": {
  "enabled": true,
  "base_url": "https://yourcompany.atlassian.net",
  "email": "you@example.com",
  "token": "your_api_token",
  "project": "ENG"
}
```

### Notion

1. Go to [https://www.notion.so/my-integrations](https://www.notion.so/my-integrations) → **New integration**
2. Give it a name, select your workspace, submit
3. Copy the **Internal Integration Token** (starts with `secret_`)
4. Open the Notion database you want to show → **⋯ menu** → **Add connections** → select your integration
5. Copy the database ID from the URL: `https://notion.so/your-workspace/DATABASE_ID?v=...`
6. Add to config:
```json
"notion": {
  "enabled": true,
  "token": "secret_your_token_here",
  "database_id": "your_database_id"
}
```

### AI Morning Summary (optional)

The dashboard can generate a concise natural-language briefing using Claude.

1. Get an [Anthropic API key](https://console.anthropic.com/)
2. Either export it as an environment variable:
   ```bash
   export ANTHROPIC_API_KEY=sk-ant-...
   ```
   Or add it to config:
   ```json
   "ai": {
     "enabled": true,
     "api_key": "sk-ant-your_key_here",
     "model": "claude-sonnet-4-6"
   }
   ```

### Weather city

Weather uses Open-Meteo (free, no key). To change the city:
```json
"weather": {
  "city": "London",
  "units": "metric",
  "enabled": true
}
```

## How the dashboard works

- The Go backend polls all configured integrations every 30 seconds (configurable via `server.poll_interval_seconds`)
- Updates are pushed to the browser over WebSocket — no page refresh needed
- Each section header shows a **LIVE** badge (green) when fetching real data, or **DEMO** (amber) when using sample data
- Sections without an available API show a step-by-step **Connect** prompt
- The todo list is auto-populated from signals across all sources; you can also add, complete, and delete todos manually
- The Pomodoro timer runs 25-minute work / 5-minute break cycles with browser notifications

## Terminal mode

If you prefer a terminal summary over the web dashboard:
```bash
make briefing
# Prints a colorized morning briefing to stdout, then exits
```

## Development commands

```bash
make test           # Run all tests with race detector
make test-verbose   # Run tests with -v flag
make test-coverage  # Generate coverage report (≥95% required)
make lint           # Run gofmt check + go vet
make ci             # lint + test + build (mirrors CI)
make clean          # Remove bin/ and coverage.out
```

Run a single test:
```bash
go test -run TestFunctionName ./internal/services/
```

## Project structure

```
cmd/
  briefing/     One-shot CLI terminal briefing
  server/       Long-running HTTP + WebSocket server
  tui/          Terminal UI (bubbletea)
frontend/       Next.js dashboard with AI SDK chat
internal/
  api/          HTTP handlers and WebSocket bridge
  config/       Configuration load/save
  models/       Shared data types
  services/     All integrations (weather, news, calendar, etc.)
  todo/         Persistent todo storage (JSON file)
  tui/          Terminal UI model and rendering
  websocket/    WebSocket hub for real-time push
web/
  templates/    Plain HTML dashboard (no Node.js required)
  static/       CSS and JS assets
```

## API reference

All endpoints return JSON.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/briefing` | Full briefing (all integrations) |
| GET | `/api/weather` | Weather data |
| GET | `/api/events` | Calendar events |
| GET | `/api/news` | News headlines |
| GET | `/api/emails` | Email messages |
| GET | `/api/slack` | Slack messages |
| GET | `/api/github` | GitHub notifications |
| GET | `/api/jira` | Jira tickets |
| GET | `/api/notion` | Notion pages |
| GET | `/api/todos` | Todo list |
| POST | `/api/todos` | Create a todo |
| PATCH | `/api/todos/:id` | Update a todo |
| PUT | `/api/todos/:id` | Replace a todo |
| DELETE | `/api/todos/:id` | Delete a todo |
| GET | `/api/signals` | Unified signal feed (Slack + Email + GitHub) |
| WS | `/ws` | WebSocket for real-time updates |
