package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
	"github.com/todogpt/daily-briefing/internal/services"
)

func testHub(t *testing.T) *services.Hub {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Server.DataDir = t.TempDir()
	return services.NewHub(cfg)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func testBriefing() *models.Briefing {
	now := time.Now()
	return &models.Briefing{
		GeneratedAt: now,
		Date:        now,
		Weather: &models.Weather{
			City:        "Test City",
			Temperature: 72,
			FeelsLike:   70,
			Humidity:    45,
			Description: "partly cloudy",
			WindSpeed:   8,
		},
		Events: []models.CalendarEvent{
			{ID: "e1", Title: "Standup", StartTime: now.Add(time.Hour), EndTime: now.Add(2 * time.Hour)},
			{ID: "e2", Title: "Design Review", Location: "Conf Room A", StartTime: now.Add(3 * time.Hour), EndTime: now.Add(4 * time.Hour)},
			{ID: "e3", Title: "Demo", MeetingURL: "https://meet.example.com", StartTime: now.Add(5 * time.Hour), EndTime: now.Add(6 * time.Hour)},
		},
		News: []models.NewsItem{
			{Title: "Test Headline 1", Source: "HN", PublishedAt: now},
			{Title: "Test Headline 2", Source: "HN", PublishedAt: now.Add(-time.Hour)},
		},
		UnreadEmails: []models.EmailMessage{
			{ID: "m1", Subject: "Hello", From: "alice@example.com", IsUnread: true, Date: now},
			{ID: "m2", Subject: "URGENT: Server Down", From: "ops@example.com", IsUnread: true, IsStarred: true, Date: now},
			{ID: "m3", Subject: "Read already", From: "foo@example.com", IsUnread: false, Date: now},
		},
		SlackMessages: []models.SlackMessage{
			{Channel: "#general", User: "alice", Text: "Hello there", IsUrgent: false},
			{Channel: "DM", User: "bob", Text: "Quick question", IsDM: true},
			{Channel: "#ops", User: "bot", Text: "Alert: disk usage high", IsUrgent: true},
		},
		GitHubNotifs: []models.GitHubNotification{
			{ID: "gh1", Title: "Fix memory leak", Repo: "org/api", Type: "PullRequest", Reason: "review_requested", Unread: true},
			{ID: "gh2", Title: "Stale issue", Repo: "org/api", Type: "Issue", Reason: "mention", Unread: false},
		},
		Todos: []models.TodoItem{
			{ID: "t1", Title: "Reply to Alice", Priority: models.PriorityHigh, Status: models.TodoPending, Source: "email"},
			{ID: "t2", Title: "Review PR", Priority: models.PriorityUrgent, Status: models.TodoInProgress, Source: "github"},
			{ID: "t3", Title: "Done task", Priority: models.PriorityLow, Status: models.TodoDone, Source: "manual"},
		},
	}
}

func baseModel() model {
	return model{width: 120, height: 40}
}

func loadedModel() model {
	m := baseModel()
	m.briefing = testBriefing()
	m.lastFetch = time.Now()
	return m
}

// ── Pure helper tests ─────────────────────────────────────────────────────────

func TestCountPending(t *testing.T) {
	todos := []models.TodoItem{
		{Status: models.TodoPending},
		{Status: models.TodoInProgress},
		{Status: models.TodoDone},
		{Status: models.TodoArchived},
	}
	if got := countPending(todos); got != 2 {
		t.Errorf("expected 2, got %d", got)
	}
}

func TestCountPendingEmpty(t *testing.T) {
	if got := countPending(nil); got != 0 {
		t.Errorf("expected 0 for nil slice, got %d", got)
	}
}

func TestFilterPending(t *testing.T) {
	todos := []models.TodoItem{
		{ID: "a", Status: models.TodoPending},
		{ID: "b", Status: models.TodoInProgress},
		{ID: "c", Status: models.TodoDone},
	}
	got := filterPending(todos)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].ID != "a" || got[1].ID != "b" {
		t.Errorf("wrong items returned: %v", got)
	}
}

func TestFilterPendingEmpty(t *testing.T) {
	if got := filterPending(nil); got != nil {
		t.Errorf("expected nil for nil input, got %v", got)
	}
}

func TestPadLines(t *testing.T) {
	s := padLines("a\nb", 5)
	lines := strings.Split(s, "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
}

func TestPadLinesAlreadyFull(t *testing.T) {
	s := padLines("a\nb\nc", 2)
	lines := strings.Split(s, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (no truncation), got %d", len(lines))
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"ab", 5, "ab"},
		{"x", 2, "x"},         // max <= 3: no truncation
		{"hello", 3, "hello"}, // max == 3: no truncation
		{"", 10, ""},
	}
	for _, c := range cases {
		got := truncate(c.in, c.max)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

// ── Model construction ────────────────────────────────────────────────────────

func TestNewModel(t *testing.T) {
	hub := testHub(t)
	m := newModel(hub)
	if !m.loading {
		t.Error("expected loading=true on new model")
	}
	if m.hub != hub {
		t.Error("expected hub to be set")
	}
}

func TestInitReturnsCmds(t *testing.T) {
	hub := testHub(t)
	m := newModel(hub)
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected non-nil Init cmd")
	}
}

// ── Window resize ─────────────────────────────────────────────────────────────

func TestUpdateWindowSize(t *testing.T) {
	m := baseModel()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	m2 := result.(model)
	if m2.width != 200 || m2.height != 50 {
		t.Errorf("expected 200×50, got %d×%d", m2.width, m2.height)
	}
}

// ── fetchDoneMsg ──────────────────────────────────────────────────────────────

func TestUpdateFetchDone(t *testing.T) {
	m := baseModel()
	m.loading = true
	b := testBriefing()
	result, _ := m.Update(fetchDoneMsg{briefing: b})
	m2 := result.(model)
	if m2.loading {
		t.Error("expected loading=false after fetchDone")
	}
	if m2.briefing != b {
		t.Error("expected briefing to be set")
	}
	if m2.lastFetch.IsZero() {
		t.Error("expected lastFetch to be set")
	}
}

// ── tickMsg ───────────────────────────────────────────────────────────────────

func TestUpdateTick(t *testing.T) {
	m := loadedModel()
	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Error("expected non-nil cmd after tick")
	}
}

// ── Key: quit ─────────────────────────────────────────────────────────────────

func TestKeyQuitUnknownKeyIsNoop(t *testing.T) {
	m := loadedModel()
	// An unrecognised key should be a no-op
	m2 := m.applyKey("F99")
	if m2.activeTab != m.activeTab {
		t.Error("unknown key should not change activeTab")
	}
}

// ── Key: tab navigation ───────────────────────────────────────────────────────

func TestKeyTabForward(t *testing.T) {
	m := loadedModel()
	m.activeTab = 0
	m2 := m.applyKey("tab")
	if m2.activeTab != 1 {
		t.Errorf("expected tab 1, got %d", m2.activeTab)
	}
	if m2.scroll != 0 {
		t.Error("scroll should reset on tab change")
	}
}

func TestKeyTabWraps(t *testing.T) {
	m := loadedModel()
	m.activeTab = numSections - 1
	m2 := m.applyKey("tab")
	if m2.activeTab != 0 {
		t.Errorf("expected wrap to 0, got %d", m2.activeTab)
	}
}

func TestKeyTabRight(t *testing.T) {
	m := loadedModel()
	m.activeTab = 2
	m2 := m.applyKey("right")
	if m2.activeTab != 3 {
		t.Errorf("expected 3, got %d", m2.activeTab)
	}
}

func TestKeyTabL(t *testing.T) {
	m := loadedModel()
	m.activeTab = 1
	m2 := m.applyKey("l")
	if m2.activeTab != 2 {
		t.Errorf("expected 2, got %d", m2.activeTab)
	}
}

func TestKeyShiftTabBackward(t *testing.T) {
	m := loadedModel()
	m.activeTab = 3
	m2 := m.applyKey("shift+tab")
	if m2.activeTab != 2 {
		t.Errorf("expected 2, got %d", m2.activeTab)
	}
}

func TestKeyShiftTabWraps(t *testing.T) {
	m := loadedModel()
	m.activeTab = 0
	m2 := m.applyKey("shift+tab")
	if m2.activeTab != numSections-1 {
		t.Errorf("expected %d, got %d", numSections-1, m2.activeTab)
	}
}

func TestKeyLeft(t *testing.T) {
	m := loadedModel()
	m.activeTab = 4
	m2 := m.applyKey("left")
	if m2.activeTab != 3 {
		t.Errorf("expected 3, got %d", m2.activeTab)
	}
}

func TestKeyH(t *testing.T) {
	m := loadedModel()
	m.activeTab = 5
	m2 := m.applyKey("h")
	if m2.activeTab != 4 {
		t.Errorf("expected 4, got %d", m2.activeTab)
	}
}

func TestKeyNumberJump(t *testing.T) {
	m := loadedModel()
	for _, key := range []string{"1", "2", "3", "4", "5", "6", "7"} {
		m2 := m.applyKey(key)
		want := int(key[0] - '1')
		if m2.activeTab != want {
			t.Errorf("key %q: expected tab %d, got %d", key, want, m2.activeTab)
		}
		if m2.scroll != 0 {
			t.Errorf("key %q: scroll should reset", key)
		}
	}
}

// ── Key: scroll ───────────────────────────────────────────────────────────────

func TestKeyScrollDown(t *testing.T) {
	m := loadedModel()
	m.activeTab = secNews
	m2 := m.applyKey("down")
	if m2.scroll != 1 {
		t.Errorf("expected scroll=1, got %d", m2.scroll)
	}
}

func TestKeyScrollDownJ(t *testing.T) {
	m := loadedModel()
	m.activeTab = secWeather
	m2 := m.applyKey("j")
	if m2.scroll != 1 {
		t.Errorf("expected scroll=1, got %d", m2.scroll)
	}
}

func TestKeyScrollUp(t *testing.T) {
	m := loadedModel()
	m.activeTab = secNews
	m.scroll = 5
	m2 := m.applyKey("up")
	if m2.scroll != 4 {
		t.Errorf("expected scroll=4, got %d", m2.scroll)
	}
}

func TestKeyScrollUpK(t *testing.T) {
	m := loadedModel()
	m.scroll = 3
	m2 := m.applyKey("k")
	if m2.scroll != 2 {
		t.Errorf("expected scroll=2, got %d", m2.scroll)
	}
}

func TestKeyScrollUpAtZero(t *testing.T) {
	m := loadedModel()
	m.scroll = 0
	m2 := m.applyKey("up")
	if m2.scroll != 0 {
		t.Error("scroll should not go below 0")
	}
}

// ── Key: todo navigation ──────────────────────────────────────────────────────

func TestKeyTodoDown(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	m.selectedTodo = 0
	m2 := m.applyKey("down")
	if m2.selectedTodo != 1 {
		t.Errorf("expected selectedTodo=1, got %d", m2.selectedTodo)
	}
}

func TestKeyTodoDownAtMax(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	pending := countPending(m.briefing.Todos)
	m.selectedTodo = pending - 1
	m2 := m.applyKey("down")
	if m2.selectedTodo != pending-1 {
		t.Errorf("selectedTodo should not exceed pending count: got %d", m2.selectedTodo)
	}
}

func TestKeyTodoDownNoBriefing(t *testing.T) {
	m := baseModel()
	m.activeTab = secTodos
	m2 := m.applyKey("down")
	if m2.scroll != 1 {
		t.Errorf("without briefing, down should scroll: got scroll=%d", m2.scroll)
	}
}

func TestKeyTodoUp(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	m.selectedTodo = 1
	m2 := m.applyKey("up")
	if m2.selectedTodo != 0 {
		t.Errorf("expected selectedTodo=0, got %d", m2.selectedTodo)
	}
}

func TestKeyTodoUpAtZero(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	m.selectedTodo = 0
	m2 := m.applyKey("up")
	if m2.selectedTodo != 0 {
		t.Error("selectedTodo should not go below 0")
	}
}

// ── Key: mark done ────────────────────────────────────────────────────────────

func TestKeyTodoMarkDoneSpace(t *testing.T) {
	hub := testHub(t)
	// seed the hub's todo service with our test todos
	hub.Todos.Add(models.TodoItem{ID: "t1", Title: "Task 1", Status: models.TodoPending})
	hub.Todos.Add(models.TodoItem{ID: "t2", Title: "Task 2", Status: models.TodoPending})

	m := loadedModel()
	m.hub = hub
	m.briefing.Todos = hub.Todos.List()
	m.activeTab = secTodos
	m.selectedTodo = 0

	m2 := m.applyKey(" ")
	// After completing, the briefing todos should be reloaded
	pendingAfter := countPending(m2.briefing.Todos)
	if pendingAfter != 1 {
		t.Errorf("expected 1 pending after marking done, got %d", pendingAfter)
	}
}

func TestKeyTodoMarkDoneEnter(t *testing.T) {
	hub := testHub(t)
	hub.Todos.Add(models.TodoItem{ID: "x1", Title: "Task A", Status: models.TodoPending})

	m := loadedModel()
	m.hub = hub
	m.briefing.Todos = hub.Todos.List()
	m.activeTab = secTodos
	m.selectedTodo = 0

	m2 := m.applyKey("enter")
	if countPending(m2.briefing.Todos) != 0 {
		t.Error("expected 0 pending after marking only todo done")
	}
}

func TestKeyTodoMarkDoneAdjustsCursor(t *testing.T) {
	hub := testHub(t)
	hub.Todos.Add(models.TodoItem{ID: "a", Title: "A", Status: models.TodoPending})
	hub.Todos.Add(models.TodoItem{ID: "b", Title: "B", Status: models.TodoPending})

	m := loadedModel()
	m.hub = hub
	m.briefing.Todos = hub.Todos.List()
	m.activeTab = secTodos
	m.selectedTodo = 1 // select last item

	m2 := m.applyKey(" ")
	// After completing "b", only "a" remains; cursor should be clamped to 0
	if m2.selectedTodo != 0 {
		t.Errorf("cursor should be clamped to 0, got %d", m2.selectedTodo)
	}
}

func TestKeyTodoMarkDoneNoHub(t *testing.T) {
	m := loadedModel()
	m.hub = nil
	m.activeTab = secTodos
	// Should not panic
	m2 := m.applyKey(" ")
	_ = m2
}

func TestKeyTodoMarkDoneOutOfRange(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	m.selectedTodo = 999
	m2 := m.applyKey(" ")
	_ = m2
}

func TestKeyMarkDoneNotTodosTab(t *testing.T) {
	m := loadedModel()
	m.activeTab = secNews
	m2 := m.applyKey(" ")
	if m2.activeTab != secNews {
		t.Error("space on non-todos tab should not change activeTab")
	}
}

// ── Key: refresh ──────────────────────────────────────────────────────────────

func TestKeyRefresh(t *testing.T) {
	m := loadedModel()
	m.loading = false
	m2 := m.applyKey("r")
	if !m2.loading {
		t.Error("expected loading=true after 'r'")
	}
}

// ── View: zero width ──────────────────────────────────────────────────────────

func TestViewZeroWidth(t *testing.T) {
	m := model{}
	v := m.View()
	if !strings.Contains(v, "Loading") {
		t.Errorf("expected Loading for zero-width model, got: %q", v)
	}
}

// ── View: loading state ───────────────────────────────────────────────────────

func TestViewLoadingNoBriefing(t *testing.T) {
	m := baseModel()
	m.loading = true
	v := m.viewSection(20)
	if !strings.Contains(v, "Fetching") {
		t.Errorf("expected Fetching... text, got: %q", v)
	}
}

func TestViewNoBriefingNotLoading(t *testing.T) {
	m := baseModel()
	m.loading = false
	v := m.viewSection(20)
	if !strings.Contains(v, "No data") {
		t.Errorf("expected 'No data' text, got: %q", v)
	}
}

// ── View: complete render ─────────────────────────────────────────────────────

func TestViewFull(t *testing.T) {
	m := loadedModel()
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view")
	}
}

// ── viewHeader ────────────────────────────────────────────────────────────────

func TestViewHeaderLoading(t *testing.T) {
	m := baseModel()
	m.loading = true
	h := m.viewHeader()
	if !strings.Contains(h, "Refreshing") {
		t.Errorf("expected 'Refreshing' in header, got: %q", h)
	}
}

func TestViewHeaderLastFetch(t *testing.T) {
	m := baseModel()
	m.lastFetch = time.Now()
	h := m.viewHeader()
	if !strings.Contains(h, "Updated") {
		t.Errorf("expected 'Updated' in header, got: %q", h)
	}
}

func TestViewHeaderNoFetch(t *testing.T) {
	m := baseModel()
	h := m.viewHeader()
	if strings.Contains(h, "Updated") {
		t.Error("should not show 'Updated' when lastFetch is zero")
	}
}

// ── viewTabs ──────────────────────────────────────────────────────────────────

func TestViewTabsContainsSectionNames(t *testing.T) {
	m := loadedModel()
	tabs := m.viewTabs()
	for _, name := range sectionNames {
		if !strings.Contains(tabs, name) {
			t.Errorf("tab bar missing section %q", name)
		}
	}
}

// ── badge ─────────────────────────────────────────────────────────────────────

func TestBadgeNilBriefing(t *testing.T) {
	m := baseModel()
	for i := 0; i < numSections; i++ {
		if b := m.badge(i); b != "" {
			t.Errorf("section %d: expected empty badge with nil briefing, got %q", i, b)
		}
	}
}

func TestBadgeNews(t *testing.T) {
	m := loadedModel()
	b := m.badge(secNews)
	if !strings.Contains(b, "2") {
		t.Errorf("expected news badge with count 2, got %q", b)
	}
}

func TestBadgeCalendar(t *testing.T) {
	m := loadedModel()
	b := m.badge(secCalendar)
	if !strings.Contains(b, "3") {
		t.Errorf("expected calendar badge with count 3, got %q", b)
	}
}

func TestBadgeEmail(t *testing.T) {
	m := loadedModel()
	b := m.badge(secEmail)
	if !strings.Contains(b, "2") { // 2 unread emails
		t.Errorf("expected email badge with count 2, got %q", b)
	}
}

func TestBadgeSlack(t *testing.T) {
	m := loadedModel()
	b := m.badge(secSlack)
	if !strings.Contains(b, "3") {
		t.Errorf("expected slack badge with count 3, got %q", b)
	}
}

func TestBadgeGitHub(t *testing.T) {
	m := loadedModel()
	b := m.badge(secGitHub)
	if !strings.Contains(b, "1") { // 1 unread
		t.Errorf("expected github badge with count 1, got %q", b)
	}
}

func TestBadgeTodos(t *testing.T) {
	m := loadedModel()
	b := m.badge(secTodos)
	if !strings.Contains(b, "2") { // 2 pending/in-progress
		t.Errorf("expected todos badge with count 2, got %q", b)
	}
}

func TestBadgeWeather(t *testing.T) {
	m := loadedModel()
	b := m.badge(secWeather) // no badge for weather
	if b != "" {
		t.Errorf("expected empty badge for weather, got %q", b)
	}
}

func TestBadgeEmptyNewsNoBadge(t *testing.T) {
	m := loadedModel()
	m.briefing.News = nil
	if b := m.badge(secNews); b != "" {
		t.Errorf("expected empty badge when no news, got %q", b)
	}
}

// ── viewStatusBar ─────────────────────────────────────────────────────────────

func TestViewStatusBarNonTodos(t *testing.T) {
	m := loadedModel()
	m.activeTab = secNews
	sb := m.viewStatusBar()
	if !strings.Contains(sb, "scroll") {
		t.Errorf("expected 'scroll' in status bar for non-todos, got: %q", sb)
	}
}

func TestViewStatusBarTodos(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	sb := m.viewStatusBar()
	if !strings.Contains(sb, "select") {
		t.Errorf("expected 'select' in status bar for todos, got: %q", sb)
	}
}

// ── Section renderers ─────────────────────────────────────────────────────────

func TestRenderNews(t *testing.T) {
	m := loadedModel()
	out := m.renderNews()
	if !strings.Contains(out, "Test Headline 1") {
		t.Errorf("expected headline in news, got: %q", out)
	}
}

func TestRenderNewsEmpty(t *testing.T) {
	m := loadedModel()
	m.briefing.News = nil
	out := m.renderNews()
	if !strings.Contains(out, "No news") {
		t.Errorf("expected 'No news' for empty news, got: %q", out)
	}
}

func TestRenderNewsMoreThan10(t *testing.T) {
	m := loadedModel()
	for i := 0; i < 15; i++ {
		m.briefing.News = append(m.briefing.News, models.NewsItem{
			Title: "Extra", Source: "HN", PublishedAt: time.Now(),
		})
	}
	out := m.renderNews()
	if !strings.Contains(out, "Top 10") {
		t.Errorf("expected 'Top 10' header for >10 news items, got: %q", out)
	}
}

func TestRenderWeather(t *testing.T) {
	m := loadedModel()
	out := m.renderWeather()
	if !strings.Contains(out, "Test City") {
		t.Errorf("expected city name in weather, got: %q", out)
	}
}

func TestRenderWeatherNil(t *testing.T) {
	m := loadedModel()
	m.briefing.Weather = nil
	out := m.renderWeather()
	if !strings.Contains(out, "No weather") {
		t.Errorf("expected 'No weather' text, got: %q", out)
	}
}

func TestRenderCalendar(t *testing.T) {
	m := loadedModel()
	out := m.renderCalendar()
	if !strings.Contains(out, "Standup") {
		t.Errorf("expected event title in calendar, got: %q", out)
	}
	if !strings.Contains(out, "Conf Room A") {
		t.Errorf("expected location in calendar, got: %q", out)
	}
	if !strings.Contains(out, "virtual") {
		t.Errorf("expected (virtual) for meeting URL, got: %q", out)
	}
}

func TestRenderCalendarEmpty(t *testing.T) {
	m := loadedModel()
	m.briefing.Events = nil
	out := m.renderCalendar()
	if !strings.Contains(out, "No events") {
		t.Errorf("expected 'No events', got: %q", out)
	}
}

func TestRenderEmail(t *testing.T) {
	m := loadedModel()
	out := m.renderEmail()
	if !strings.Contains(out, "Hello") {
		t.Errorf("expected email subject, got: %q", out)
	}
	if !strings.Contains(out, "★") {
		t.Errorf("expected star for starred email, got: %q", out)
	}
}

func TestRenderEmailInboxZero(t *testing.T) {
	m := loadedModel()
	for i := range m.briefing.UnreadEmails {
		m.briefing.UnreadEmails[i].IsUnread = false
	}
	out := m.renderEmail()
	if !strings.Contains(out, "Inbox zero") {
		t.Errorf("expected inbox zero, got: %q", out)
	}
}

func TestRenderSlack(t *testing.T) {
	m := loadedModel()
	out := m.renderSlack()
	if !strings.Contains(out, "#general") {
		t.Errorf("expected #general in slack output, got: %q", out)
	}
	if !strings.Contains(out, "@") {
		t.Errorf("expected DM marker '@', got: %q", out)
	}
	if !strings.Contains(out, "!") {
		t.Errorf("expected urgent marker '!', got: %q", out)
	}
}

func TestRenderSlackEmpty(t *testing.T) {
	m := loadedModel()
	m.briefing.SlackMessages = nil
	out := m.renderSlack()
	if !strings.Contains(out, "No Slack") {
		t.Errorf("expected 'No Slack messages', got: %q", out)
	}
}

func TestRenderGitHub(t *testing.T) {
	m := loadedModel()
	out := m.renderGitHub()
	if !strings.Contains(out, "Fix memory leak") {
		t.Errorf("expected PR title, got: %q", out)
	}
}

func TestRenderGitHubIssueType(t *testing.T) {
	m := loadedModel()
	m.briefing.GitHubNotifs = []models.GitHubNotification{
		{ID: "i1", Title: "Open Issue", Repo: "org/x", Type: "Issue", Reason: "mention", Unread: true},
	}
	out := m.renderGitHub()
	if !strings.Contains(out, "[IS]") {
		t.Errorf("expected [IS] for issue type, got: %q", out)
	}
}

func TestRenderGitHubNoUnread(t *testing.T) {
	m := loadedModel()
	for i := range m.briefing.GitHubNotifs {
		m.briefing.GitHubNotifs[i].Unread = false
	}
	out := m.renderGitHub()
	if !strings.Contains(out, "No unread") {
		t.Errorf("expected 'No unread', got: %q", out)
	}
}

func TestRenderTodos(t *testing.T) {
	m := loadedModel()
	out := m.renderTodos()
	if !strings.Contains(out, "Reply to Alice") {
		t.Errorf("expected todo title, got: %q", out)
	}
	if !strings.Contains(out, "[urgent]") {
		t.Errorf("expected urgent priority label, got: %q", out)
	}
}

func TestRenderTodosAllDone(t *testing.T) {
	m := loadedModel()
	for i := range m.briefing.Todos {
		m.briefing.Todos[i].Status = models.TodoDone
	}
	out := m.renderTodos()
	if !strings.Contains(out, "All done") {
		t.Errorf("expected 'All done', got: %q", out)
	}
}

func TestRenderTodosPriorityColors(t *testing.T) {
	m := loadedModel()
	m.briefing.Todos = []models.TodoItem{
		{ID: "a", Title: "Low", Priority: models.PriorityLow, Status: models.TodoPending},
		{ID: "b", Title: "Med", Priority: models.PriorityMedium, Status: models.TodoPending},
	}
	out := m.renderTodos()
	if !strings.Contains(out, "Low") || !strings.Contains(out, "Med") {
		t.Error("expected todo titles in output")
	}
}

// ── viewSection scroll ────────────────────────────────────────────────────────

func TestViewSectionScrollOffset(t *testing.T) {
	m := loadedModel()
	m.activeTab = secNews
	// With scroll=0, first line includes the title
	s0 := m.viewSection(30)
	m.scroll = 3
	s3 := m.viewSection(30)
	if s0 == s3 {
		t.Error("scrolled content should differ from non-scrolled")
	}
}

func TestViewSectionScrollPastEnd(t *testing.T) {
	m := loadedModel()
	m.activeTab = secNews
	m.scroll = 9999
	// Should not panic; should just show empty lines
	s := m.viewSection(10)
	if s == "" {
		t.Error("expected non-empty string even when scrolled past end")
	}
}

// ── Full section coverage via viewSection ─────────────────────────────────────

func TestViewSectionAllTabs(t *testing.T) {
	m := loadedModel()
	for i := 0; i < numSections; i++ {
		m.activeTab = i
		s := m.viewSection(20)
		if s == "" {
			t.Errorf("section %d returned empty string", i)
		}
	}
}

// ── Update with KeyMsg via tea.KeyMsg ─────────────────────────────────────────

func TestUpdateKeyMsgQ(t *testing.T) {
	m := loadedModel()
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m2 := result.(model)
	// Quit should return a Quit cmd
	if cmd == nil {
		t.Error("expected quit cmd")
	}
	_ = m2
}

func TestUpdateKeyMsgCtrlC(t *testing.T) {
	m := loadedModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	// ctrl+c must produce a Quit cmd — verify it is non-nil
	if cmd == nil {
		t.Error("expected Quit cmd for ctrl+c")
	}
}

// TestUpdateUnknownMsg ensures unrecognised message types are handled safely.
func TestUpdateUnknownMsg(t *testing.T) {
	m := loadedModel()
	result, cmd := m.Update("some random string message")
	_ = result.(model)
	if cmd != nil {
		t.Error("expected nil cmd for unknown msg type")
	}
}

// ── doFetch execution ─────────────────────────────────────────────────────────

func TestDoFetch(t *testing.T) {
	hub := testHub(t)
	cmd := doFetch(hub)
	if cmd == nil {
		t.Fatal("doFetch should return a non-nil Cmd")
	}
	// Execute the cmd — it performs a FetchAll and returns a fetchDoneMsg
	msg := cmd()
	result, ok := msg.(fetchDoneMsg)
	if !ok {
		t.Fatalf("expected fetchDoneMsg, got %T", msg)
	}
	if result.briefing == nil {
		t.Error("expected non-nil briefing from doFetch cmd")
	}
}

// ── View edge: tiny terminal ──────────────────────────────────────────────────

func TestViewContentHeightFloor(t *testing.T) {
	m := loadedModel()
	// Extremely small height forces the contentH < 1 branch
	m.height = 2
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view even with tiny height")
	}
}

func TestViewHeaderGapNegative(t *testing.T) {
	m := loadedModel()
	// Very narrow width forces gap < 0
	m.width = 5
	h := m.viewHeader()
	if h == "" {
		t.Error("expected non-empty header even with tiny width")
	}
}

// ── scheduleTickCmd non-nil ───────────────────────────────────────────────────

func TestScheduleTick(t *testing.T) {
	cmd := scheduleTick()
	if cmd == nil {
		t.Error("scheduleTick should return a non-nil Cmd")
	}
}

// ── Key: new todo (n) ─────────────────────────────────────────────────────────

func TestKeyNEntersInputMode(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	m2 := m.applyKey("n")
	if !m2.inputMode {
		t.Error("expected inputMode=true after 'n'")
	}
	if m2.inputText != "" {
		t.Error("expected empty inputText on entry")
	}
}

func TestKeyNOnlyWorksOnTodosTab(t *testing.T) {
	m := loadedModel()
	m.activeTab = secNews
	m2 := m.applyKey("n")
	if m2.inputMode {
		t.Error("'n' should not enter input mode on non-todos tab")
	}
}

func TestInputModeTyping(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	m.inputMode = true
	m.inputText = ""
	for _, ch := range "hello" {
		m = m.applyInputKey(string(ch))
	}
	if m.inputText != "hello" {
		t.Errorf("expected inputText='hello', got %q", m.inputText)
	}
}

func TestInputModeBackspace(t *testing.T) {
	m := loadedModel()
	m.inputMode = true
	m.inputText = "abc"
	m2 := m.applyInputKey("backspace")
	if m2.inputText != "ab" {
		t.Errorf("expected 'ab' after backspace, got %q", m2.inputText)
	}
}

func TestInputModeBackspaceEmpty(t *testing.T) {
	m := loadedModel()
	m.inputMode = true
	m.inputText = ""
	m2 := m.applyInputKey("backspace")
	if m2.inputText != "" {
		t.Error("backspace on empty should not panic")
	}
}

func TestInputModeEscCancels(t *testing.T) {
	m := loadedModel()
	m.inputMode = true
	m.inputText = "unfinished"
	m2 := m.applyInputKey("esc")
	if m2.inputMode {
		t.Error("expected inputMode=false after esc")
	}
	if m2.inputText != "" {
		t.Error("expected inputText cleared after esc")
	}
}

func TestInputModeEnterCreatesTodo(t *testing.T) {
	hub := testHub(t)
	m := loadedModel()
	m.hub = hub
	m.activeTab = secTodos
	m.inputMode = true
	m.inputText = "My new task"

	m2 := m.applyInputKey("enter")
	if m2.inputMode {
		t.Error("expected inputMode=false after enter")
	}
	if m2.inputText != "" {
		t.Error("expected inputText cleared after enter")
	}
	found := false
	for _, todo := range hub.Todos.List() {
		if todo.Title == "My new task" {
			found = true
		}
	}
	if !found {
		t.Error("expected new todo to be added to hub")
	}
}

func TestInputModeEnterEmptyDoesNotCreate(t *testing.T) {
	hub := testHub(t)
	before := len(hub.Todos.List())
	m := loadedModel()
	m.hub = hub
	m.inputMode = true
	m.inputText = ""
	m.applyInputKey("enter")
	if len(hub.Todos.List()) != before {
		t.Error("empty input should not create a todo")
	}
}

func TestInputModeEnterNoHub(t *testing.T) {
	m := loadedModel()
	m.hub = nil
	m.inputMode = true
	m.inputText = "task"
	m2 := m.applyInputKey("enter")
	// Should not panic
	if m2.inputMode {
		t.Error("expected inputMode=false even with nil hub")
	}
}

func TestInputModeMultiRuneKeyIgnored(t *testing.T) {
	m := loadedModel()
	m.inputMode = true
	m.inputText = "x"
	m2 := m.applyInputKey("ctrl+a") // multi-char key — should be ignored
	if m2.inputText != "x" {
		t.Errorf("multi-rune key should be ignored, got %q", m2.inputText)
	}
}

func TestUpdateKeyInInputMode(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	m.inputMode = true
	m.inputText = ""
	// 'q' in input mode should NOT quit — it should type 'q'
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m2 := result.(model)
	if cmd != nil {
		t.Error("'q' in input mode should not produce quit cmd")
	}
	if m2.inputText != "q" {
		t.Errorf("expected inputText='q', got %q", m2.inputText)
	}
}

func TestUpdateCtrlCInInputModeQuits(t *testing.T) {
	m := loadedModel()
	m.inputMode = true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c should still quit even in input mode")
	}
}

// ── Key: delete todo (d) ──────────────────────────────────────────────────────

func TestKeyDeleteTodo(t *testing.T) {
	hub := testHub(t)
	hub.Todos.Add(models.TodoItem{ID: "del1", Title: "Delete me", Status: models.TodoPending})
	hub.Todos.Add(models.TodoItem{ID: "del2", Title: "Keep me", Status: models.TodoPending})

	m := loadedModel()
	m.hub = hub
	m.briefing.Todos = hub.Todos.List()
	m.activeTab = secTodos
	m.selectedTodo = 0

	m2 := m.applyKey("d")
	if countPending(m2.briefing.Todos) != 1 {
		t.Errorf("expected 1 pending after delete, got %d", countPending(m2.briefing.Todos))
	}
}

func TestKeyDeleteNoHub(t *testing.T) {
	m := loadedModel()
	m.hub = nil
	m.activeTab = secTodos
	m2 := m.applyKey("d")
	_ = m2 // should not panic
}

func TestKeyDeleteNotTodosTab(t *testing.T) {
	m := loadedModel()
	m.activeTab = secNews
	m2 := m.applyKey("d")
	if m2.activeTab != secNews {
		t.Error("delete on non-todos tab should not change tab")
	}
}

func TestKeyDeleteClampsSelection(t *testing.T) {
	hub := testHub(t)
	hub.Todos.Add(models.TodoItem{ID: "a", Title: "A", Status: models.TodoPending})
	hub.Todos.Add(models.TodoItem{ID: "b", Title: "B", Status: models.TodoPending})

	m := loadedModel()
	m.hub = hub
	m.briefing.Todos = hub.Todos.List()
	m.activeTab = secTodos
	m.selectedTodo = 1

	m2 := m.applyKey("d")
	if m2.selectedTodo != 0 {
		t.Errorf("cursor should clamp to 0 after deleting last item, got %d", m2.selectedTodo)
	}
}

// ── Key: in-progress toggle (i) ───────────────────────────────────────────────

func TestKeyInProgressToggle(t *testing.T) {
	hub := testHub(t)
	hub.Todos.Add(models.TodoItem{ID: "ip1", Title: "Task", Status: models.TodoPending})

	m := loadedModel()
	m.hub = hub
	m.briefing.Todos = hub.Todos.List()
	m.activeTab = secTodos
	m.selectedTodo = 0

	m2 := m.applyKey("i")
	items := m2.briefing.Todos
	var found *models.TodoItem
	for i := range items {
		if items[i].ID == "ip1" {
			found = &items[i]
		}
	}
	if found == nil {
		t.Fatal("todo not found after toggle")
	}
	if found.Status != models.TodoInProgress {
		t.Errorf("expected TodoInProgress after first toggle, got %v", found.Status)
	}

	// Toggle back to pending
	m3 := m2.applyKey("i")
	items3 := m3.briefing.Todos
	for i := range items3 {
		if items3[i].ID == "ip1" && items3[i].Status != models.TodoPending {
			t.Errorf("expected TodoPending after second toggle, got %v", items3[i].Status)
		}
	}
}

func TestKeyInProgressNoHub(t *testing.T) {
	m := loadedModel()
	m.hub = nil
	m.activeTab = secTodos
	m2 := m.applyKey("i")
	_ = m2
}

// ── Render: input mode ────────────────────────────────────────────────────────

func TestRenderTodosInputMode(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	m.inputMode = true
	m.inputText = "Buy bread"
	out := m.renderTodos()
	if !strings.Contains(out, "Buy bread") {
		t.Errorf("expected input text in render, got: %q", out)
	}
	if !strings.Contains(out, "New todo") {
		t.Errorf("expected 'New todo' header, got: %q", out)
	}
}

func TestRenderTodosInputModeEmpty(t *testing.T) {
	m := loadedModel()
	m.briefing.Todos = nil
	m.inputMode = true
	out := m.renderTodos()
	// Should show prompt even when no todos
	if !strings.Contains(out, "New todo") {
		t.Errorf("expected prompt even with empty todos, got: %q", out)
	}
}

func TestRenderTodosInProgressMarker(t *testing.T) {
	m := loadedModel()
	m.briefing.Todos = []models.TodoItem{
		{ID: "x", Title: "Working on it", Status: models.TodoInProgress, Priority: models.PriorityMedium, Source: "manual"},
	}
	out := m.renderTodos()
	if !strings.Contains(out, "●") {
		t.Errorf("expected in-progress marker ●, got: %q", out)
	}
}

func TestViewStatusBarInputMode(t *testing.T) {
	m := loadedModel()
	m.inputMode = true
	sb := m.viewStatusBar()
	if !strings.Contains(sb, "confirm") {
		t.Errorf("expected 'confirm' in input mode status bar, got: %q", sb)
	}
	if strings.Contains(sb, "quit") {
		t.Errorf("should not show 'quit' in input mode status bar")
	}
}

func TestViewStatusBarTodosActions(t *testing.T) {
	m := loadedModel()
	m.activeTab = secTodos
	sb := m.viewStatusBar()
	if !strings.Contains(sb, "new") {
		t.Errorf("expected 'new' in todos status bar, got: %q", sb)
	}
	if !strings.Contains(sb, "delete") {
		t.Errorf("expected 'delete' in todos status bar, got: %q", sb)
	}
}
