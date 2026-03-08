# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make run          # Build and start the web dashboard (http://localhost:8080)
make briefing     # Build and print a one-shot terminal briefing
make tui          # Build and launch the interactive terminal UI
make test         # Run all tests with race detector
make test-verbose # Run tests with -v flag
make test-coverage # Generate coverage report; fails if total < 95%
make lint         # Run gofmt check + go vet
make ci           # lint + test-coverage + build (full local CI gate)
make init         # Generate default config at ~/.daily-briefing/config.json
make clean        # Remove bin/ and coverage.out
```

Security & audit:
```bash
make security     # Static analysis with gosec (auto-installs if missing)
make vuln         # Vulnerability scan with govulncheck (auto-installs if missing)
make privacy      # Grep-based scan: hardcoded secrets, file perms, omitempty on creds
make audit        # security + vuln + privacy (full audit suite)
```

Run a single test:
```bash
go test -run TestFunctionName ./internal/services/
go test -run TestFunctionName ./internal/...
```

Build only (outputs to `bin/`):
```bash
go build -o bin/briefing ./cmd/briefing
go build -o bin/server   ./cmd/server
go build -o bin/tui      ./cmd/tui
```

## Architecture

The app has three binaries sharing the same `internal/` packages:

- **`cmd/briefing`** — one-shot CLI that fetches all data and prints a colorized terminal summary, then exits
- **`cmd/server`** — long-running HTTP server with WebSocket push, serves the web dashboard at `/`
- **`cmd/tui`** — interactive bubbletea TUI dashboard (tabs: News, Weather, Calendar, Email, Slack, GitHub, Todos)

### Data flow

1. `services.Hub` owns all service instances (`WeatherService`, `NewsService`, `CalendarService`, `SlackService`, `EmailService`, `GitHubService`, `JiraService`, `NotionService`, `TodoService`).
2. `hub.FetchAll()` fetches all services in parallel (via `sync.WaitGroup`), then calls `TodoService.GenerateFromBriefing()` which auto-creates todos from signals.
3. The server calls `hub.StartPolling()` in a goroutine, which runs `FetchAll()` on a configurable interval and broadcasts `DashboardUpdate` structs to subscribers.
4. `api.Server` subscribes to the hub and forwards updates over WebSocket to browser clients via `websocket.Hub`.

### Key design points

- **`TodoService` persists to `~/.daily-briefing/todos.json`** via `todo.Store`. `NewHub` calls `newTodoService(cfg)` which attaches a `todo.Store`; falls back to in-memory if the store cannot be created. **Test isolation**: pass `cfg.Server.DataDir = t.TempDir()` before calling `services.NewHub(cfg)` so tests don't touch the real store. Both `services/hub_test.go` (`testConfig()`), `services/hub_extra_test.go`, `tui/tui_test.go` (`testHub(t)`), and `api/server_test.go` (`testServer(t)`) follow this pattern.
- **`TodoService.GenerateFromBriefing`** deduplicates using a `seen` map keyed by `"source:id"`. Re-fetches won't add duplicate todos within a process lifetime. After restart the `seen` map is rebuilt from loaded persisted items. `GenerateFromBriefing` generates todos from Email, Slack, GitHub, Jira, Notion, and Calendar signals.
- All services have a `GetCached()` method returning the last successful fetch result, and a `Fetch()` method that hits the real API (or returns mock data when unconfigured).
- Config lives at `~/.daily-briefing/config.json`. Each service section has an `enabled` bool. Services fall back to mock/demo data when disabled or when API keys are missing.
- The module path is `github.com/todogpt/daily-briefing`. Key dependencies: `github.com/gorilla/websocket`, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`.
- **Security**: `api.Server.Start()` uses `http.Server` with 15s read/write and 60s idle timeouts (not bare `ListenAndServe`). Config dir created with `0750`, files with `0600`. All credential fields in config structs carry `omitempty`. Package-level URL vars (`openMeteoBaseURL`, `geocodingBaseURL`, `hackerNewsBaseURL`, `githubAPIBaseURL`, `jiraAPIPath`, `notionAPIBaseURL`) are overridable in tests via httptest servers.
- **TUI** (`cmd/tui`): bubbletea model with `applyKey` (pure, testable). Key bindings: tabs (←/→/1-7), scroll (↑/↓), todo select/done (space/enter), in-progress toggle (i), delete (d), new todo (n+enter), refresh (r), quit (q), pomodoro start/stop (p). Pomodoro timer config: `pomodoro.work_minutes`, `pomodoro.break_minutes`, `pomodoro.enabled`.
- **Real APIs**: GitHub (`/notifications`), Jira (`/rest/api/3/search` basic auth), Notion (`/v1/databases/{id}/query` bearer). All use cache→mock fallback chain on error.

### Linting

`golangci-lint` is configured in `.golangci.yml` with: `errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`, `gocritic`. Errors from `fmt.Printf`/`fmt.Println`/`http.ResponseWriter.Write` are excluded. Test files are excluded from `errcheck`.

## Git workflow

**NEVER commit directly to `main`.** Every change — no matter how small — must go through a separate branch, commit, and PR. Use the squash merge method when merging via the GitHub MCP tool.

Branch naming: `claude/<short-description>` (e.g. `claude/tui-security-audit-coverage`)

Standard flow:
```bash
git checkout -b claude/<description>
# make changes, run make ci && make audit
git add <files>
git commit -m "..."
git push -u origin claude/<description>
# create PR via GitHub MCP tool (owner: sameenai, repo: TODOGPT)
# merge via GitHub MCP tool with squash method
git checkout main && git pull origin main
git branch -d claude/<description>
git push origin --delete claude/<description>
```

### Commit quality standards

- **Always run `make ci` before opening a PR.** For security-relevant changes also run `make audit`. The PR must not be created if CI fails.
- **Commits must be modular, small, and digestible** — each commit should represent one logical change that can be reviewed, tested, and reverted in isolation. Avoid bundling unrelated changes.
- **Tests must accompany every change** that adds or modifies behaviour. New Go code requires tests that keep coverage at ≥ 95%. New frontend components should have at minimum happy-path and error-state tests.
- **High code quality is non-negotiable**: no dead code, no silenced linter warnings without a comment explaining why, no unchecked errors in hot paths.

Update `CLAUDE.md` whenever new commands, architectural patterns, or workflow conventions are established.
