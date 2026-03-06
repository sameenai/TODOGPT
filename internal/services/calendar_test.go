package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

func TestNewCalendarService(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestCalendarIsLiveNoURL(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	if svc.IsLive() {
		t.Error("expected IsLive=false when no ICalURL configured")
	}
}

func TestCalendarIsLiveWithURL(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{ICalURL: "http://example.com/cal.ics"})
	if !svc.IsLive() {
		t.Error("expected IsLive=true when ICalURL is set")
	}
}

func TestCalendarFetchNoURLReturnsEmpty(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	events, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected no events when no ICalURL, got %d", len(events))
	}
}

func TestCalendarGetCachedEmpty(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	cached := svc.GetCached()
	if cached != nil {
		t.Errorf("expected nil cache before any fetch, got %v", cached)
	}
}

func icalFeedWithTodayEvent(title, location string) string {
	// Use time.Now() UTC for both start and end to guarantee the event
	// falls within today regardless of the hour (avoids %24 wrap-around bug).
	start := time.Now().UTC()
	end := start.Add(time.Hour)
	dtStart := start.Format("20060102T150405Z")
	dtEnd := end.Format("20060102T150405Z")
	return "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:test-uid-1\r\n" +
		"SUMMARY:" + title + "\r\n" +
		"DTSTART:" + dtStart + "\r\n" +
		"DTEND:" + dtEnd + "\r\n" +
		"LOCATION:" + location + "\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"
}

func TestCalendarFetchRealICalFeed(t *testing.T) {
	feed := icalFeedWithTodayEvent("Standup", "Zoom")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		fmt.Fprint(w, feed)
	}))
	defer srv.Close()

	svc := NewCalendarService(config.GoogleConfig{ICalURL: srv.URL})
	events, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Standup" {
		t.Errorf("expected title Standup, got %q", events[0].Title)
	}
	if events[0].Location != "Zoom" {
		t.Errorf("expected location Zoom, got %q", events[0].Location)
	}
	if events[0].Source != "ical" {
		t.Errorf("expected source ical, got %q", events[0].Source)
	}
}

func TestCalendarFetchCachesResult(t *testing.T) {
	feed := icalFeedWithTodayEvent("CachedEvent", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, feed)
	}))
	defer srv.Close()

	svc := NewCalendarService(config.GoogleConfig{ICalURL: srv.URL})
	svc.Fetch()
	cached := svc.GetCached()
	if len(cached) != 1 {
		t.Errorf("expected 1 cached event, got %d", len(cached))
	}
}

func TestCalendarFetchServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := NewCalendarService(config.GoogleConfig{ICalURL: srv.URL})
	_, err := svc.Fetch()
	if err == nil {
		t.Error("expected error on HTTP 500")
	}
}

func TestParseICalTodayEventsFiltersOld(t *testing.T) {
	// Use 48 hours ago so the event (+ 1h duration) never overlaps today,
	// even when running near midnight UTC.
	twoDaysAgo := time.Now().Add(-48 * time.Hour).UTC()
	feed := fmt.Sprintf("BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nSUMMARY:Old\r\n"+
		"DTSTART:%s\r\nDTEND:%s\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n",
		twoDaysAgo.Format("20060102T150405Z"),
		twoDaysAgo.Add(time.Hour).Format("20060102T150405Z"),
	)
	events, err := parseICalTodayEvents(strings.NewReader(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for yesterday, got %d", len(events))
	}
}

func TestParseICalAllDayEvent(t *testing.T) {
	today := time.Now().Format("20060102")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("20060102")
	feed := fmt.Sprintf("BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nSUMMARY:Holiday\r\n"+
		"DTSTART;VALUE=DATE:%s\r\nDTEND;VALUE=DATE:%s\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n",
		today, tomorrow,
	)
	events, err := parseICalTodayEvents(strings.NewReader(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 all-day event, got %d", len(events))
	}
	if !events[0].AllDay {
		t.Error("expected AllDay=true")
	}
}

func TestParseICalTextUnescaping(t *testing.T) {
	now := time.Now().UTC()
	feed := fmt.Sprintf("BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\n"+
		"SUMMARY:Team\\, All Hands\r\n"+
		"DTSTART:%s\r\nDTEND:%s\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n",
		now.Format("20060102T150405Z"),
		now.Add(time.Hour).Format("20060102T150405Z"),
	)
	events, err := parseICalTodayEvents(strings.NewReader(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 1 && events[0].Title != "Team, All Hands" {
		t.Errorf("expected unescaped title, got %q", events[0].Title)
	}
}

func TestParseICalLineFolding(t *testing.T) {
	now := time.Now().UTC()
	// Line folded: SUMMARY split across two lines with SP continuation
	feed := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\n" +
		"SUMMARY:Long Meeting\r\n" +
		" Title\r\n" +
		fmt.Sprintf("DTSTART:%s\r\n", now.Format("20060102T150405Z")) +
		fmt.Sprintf("DTEND:%s\r\n", now.Add(time.Hour).Format("20060102T150405Z")) +
		"END:VEVENT\r\nEND:VCALENDAR\r\n"
	events, err := parseICalTodayEvents(strings.NewReader(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 1 && events[0].Title != "Long MeetingTitle" {
		t.Errorf("expected unfolded title, got %q", events[0].Title)
	}
}

func TestSplitICalLine(t *testing.T) {
	name, val, params := splitICalLine("DTSTART;TZID=America/New_York:20240101T090000")
	if name != "DTSTART" {
		t.Errorf("expected DTSTART, got %q", name)
	}
	if val != "20240101T090000" {
		t.Errorf("expected datetime value, got %q", val)
	}
	if params["TZID"] != "America/New_York" {
		t.Errorf("expected TZID param, got %q", params["TZID"])
	}
}

func TestParseICalDateTimeUTC(t *testing.T) {
	ts, allDay := parseICalDateTime("20240115T143000Z", nil)
	if ts.IsZero() {
		t.Error("expected non-zero time")
	}
	if allDay {
		t.Error("expected allDay=false")
	}
}

func TestParseICalDateTimeAllDay(t *testing.T) {
	ts, allDay := parseICalDateTime("20240115", map[string]string{"VALUE": "DATE"})
	if ts.IsZero() {
		t.Error("expected non-zero time")
	}
	if !allDay {
		t.Error("expected allDay=true")
	}
}

func TestCalendarFetchFallsBackToCache(t *testing.T) {
	// First fetch from a good server to populate cache.
	feed := icalFeedWithTodayEvent("Cached", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, feed)
	}))
	svc := NewCalendarService(config.GoogleConfig{ICalURL: srv.URL})
	svc.Fetch()
	srv.Close() // server down — next fetch should return cache

	events, err := svc.Fetch()
	if err != nil {
		t.Fatalf("expected cache fallback, got error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 cached event, got %d", len(events))
	}
}

func TestParseICalDateTimeLocalTime(t *testing.T) {
	// Local time without Z suffix and without TZID
	ts, allDay := parseICalDateTime("20240115T143000", nil)
	if ts.IsZero() {
		t.Error("expected non-zero time for local datetime")
	}
	if allDay {
		t.Error("expected allDay=false for datetime")
	}
}

func TestParseICalDateTimeWithTZID(t *testing.T) {
	ts, allDay := parseICalDateTime("20240115T090000", map[string]string{"TZID": "America/New_York"})
	if ts.IsZero() {
		t.Error("expected non-zero time with TZID")
	}
	if allDay {
		t.Error("expected allDay=false")
	}
}

func TestParseICalDateTimeInvalid(t *testing.T) {
	ts, _ := parseICalDateTime("not-a-date", nil)
	if !ts.IsZero() {
		t.Error("expected zero time for invalid input")
	}
}

func TestSplitICalLineNoColon(t *testing.T) {
	name, val, _ := splitICalLine("BEGINVCALENDAR")
	if name != "BEGINVCALENDAR" {
		t.Errorf("expected full string as name, got %q", name)
	}
	if val != "" {
		t.Errorf("expected empty value, got %q", val)
	}
}

func TestParseICalEventMissingUID(t *testing.T) {
	now := time.Now().UTC()
	feed := fmt.Sprintf("BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\n"+
		"SUMMARY:No UID Event\r\n"+
		"DTSTART:%s\r\nDTEND:%s\r\n"+
		"END:VEVENT\r\nEND:VCALENDAR\r\n",
		now.Format("20060102T150405Z"),
		now.Add(time.Hour).Format("20060102T150405Z"),
	)
	events, err := parseICalTodayEvents(strings.NewReader(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "No UID Event" {
		t.Errorf("unexpected title: %q", events[0].Title)
	}
}

func TestParseICalEventNoEndTime(t *testing.T) {
	// VEVENT with no DTEND — should default to start + 1h
	now := time.Now().UTC()
	feed := fmt.Sprintf("BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\n"+
		"SUMMARY:No End\r\nDTSTART:%s\r\n"+
		"END:VEVENT\r\nEND:VCALENDAR\r\n",
		now.Format("20060102T150405Z"),
	)
	events, err := parseICalTodayEvents(strings.NewReader(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event with no DTEND, got %d", len(events))
	}
}

func TestParseICalEventNoStartTime(t *testing.T) {
	// VEVENT with no DTSTART — should be skipped
	feed := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nSUMMARY:No Start\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	events, err := parseICalTodayEvents(strings.NewReader(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events when no DTSTART, got %d", len(events))
	}
}

func TestParseICalEventWithURL(t *testing.T) {
	now := time.Now().UTC()
	feed := fmt.Sprintf("BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\n"+
		"SUMMARY:Meeting\r\nDTSTART:%s\r\nDTEND:%s\r\n"+
		"URL:https://meet.example.com/abc\r\n"+
		"END:VEVENT\r\nEND:VCALENDAR\r\n",
		now.Format("20060102T150405Z"),
		now.Add(time.Hour).Format("20060102T150405Z"),
	)
	events, err := parseICalTodayEvents(strings.NewReader(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].MeetingURL != "https://meet.example.com/abc" {
		t.Errorf("unexpected MeetingURL: %q", events[0].MeetingURL)
	}
}

func TestParseICalDateTimeUnknownTZID(t *testing.T) {
	// Unknown TZID falls back to local time
	ts, allDay := parseICalDateTime("20240115T090000", map[string]string{"TZID": "Invalid/Timezone"})
	if ts.IsZero() {
		t.Error("expected non-zero time even with unknown TZID")
	}
	if allDay {
		t.Error("expected allDay=false")
	}
}

func TestParseICalDateTimeAllDayNoParam(t *testing.T) {
	// 8-char value without T is treated as all-day
	ts, allDay := parseICalDateTime("20240115", nil)
	if ts.IsZero() {
		t.Error("expected non-zero time for date-only value")
	}
	if !allDay {
		t.Error("expected allDay=true for date-only value")
	}
}

func TestUnescapeICalText(t *testing.T) {
	cases := []struct{ in, want string }{
		{`hello\nworld`, "hello\nworld"},
		{`foo\,bar`, "foo,bar"},
		{`a\\b`, `a\b`},
	}
	for _, c := range cases {
		got := unescapeICalText(c.in)
		if got != c.want {
			t.Errorf("unescapeICalText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── Google Calendar API tests ──────────────────────────────────────────────────

func gcalResponse(items string) string {
	return `{"kind":"calendar#events","items":[` + items + `]}`
}

func gcalItem(id, summary, start, end string) string {
	return fmt.Sprintf(`{"id":%q,"summary":%q,"start":{"dateTime":%q},"end":{"dateTime":%q},"hangoutLink":"https://meet.google.com/abc"}`, id, summary, start, end)
}

func TestFetchGoogleCalendarSuccess(t *testing.T) {
	now := time.Now()
	start := now.Add(time.Hour).UTC().Format(time.RFC3339)
	end := now.Add(2 * time.Hour).UTC().Format(time.RFC3339)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, gcalResponse(gcalItem("evt1", "Team Standup", start, end)))
	}))
	defer srv.Close()

	orig := googleCalendarBaseURL
	googleCalendarBaseURL = srv.URL
	defer func() { googleCalendarBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	// Inject a valid token so auth.Client() returns a real client
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewCalendarServiceWithAuth(config.GoogleConfig{}, auth)
	events, err := svc.fetchGoogleCalendar(auth.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Team Standup" {
		t.Errorf("expected title 'Team Standup', got %q", events[0].Title)
	}
	if events[0].MeetingURL == "" {
		t.Error("expected MeetingURL to be set")
	}
}

func TestFetchGoogleCalendarAllDay(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	itemJSON := fmt.Sprintf(`{"id":"ad1","summary":"All Day","start":{"date":%q},"end":{"date":%q}}`, today, tomorrow)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, gcalResponse(itemJSON))
	}))
	defer srv.Close()

	orig := googleCalendarBaseURL
	googleCalendarBaseURL = srv.URL
	defer func() { googleCalendarBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	events, err := svc_calendar(auth).fetchGoogleCalendar(auth.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least 1 event")
	}
	if !events[0].AllDay {
		t.Error("expected AllDay=true")
	}
}

func TestFetchGoogleCalendarAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"error":{"code":401,"message":"Unauthorized"}}`)
	}))
	defer srv.Close()

	orig := googleCalendarBaseURL
	googleCalendarBaseURL = srv.URL
	defer func() { googleCalendarBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	_, err := svc_calendar(auth).fetchGoogleCalendar(auth.Client())
	if err == nil {
		t.Error("expected error for API error response")
	}
}

func TestFetchGoogleCalendarNilClient(t *testing.T) {
	svc := NewCalendarService(config.GoogleConfig{})
	_, err := svc.fetchGoogleCalendar(nil)
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestFetchGoogleCalendarInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not-json")
	}))
	defer srv.Close()

	orig := googleCalendarBaseURL
	googleCalendarBaseURL = srv.URL
	defer func() { googleCalendarBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	_, err := svc_calendar(auth).fetchGoogleCalendar(auth.Client())
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCalendarFetchPrefersGoogleAPI(t *testing.T) {
	now := time.Now()
	start := now.Add(time.Hour).UTC().Format(time.RFC3339)
	end := now.Add(2 * time.Hour).UTC().Format(time.RFC3339)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, gcalResponse(gcalItem("e1", "Google Event", start, end)))
	}))
	defer srv.Close()

	orig := googleCalendarBaseURL
	googleCalendarBaseURL = srv.URL
	defer func() { googleCalendarBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewCalendarServiceWithAuth(config.GoogleConfig{}, auth)
	events, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected events from Google Calendar API")
	}
	if events[0].Source != "google" {
		t.Errorf("expected source='google', got %q", events[0].Source)
	}
}

func TestFetchGoogleCalendarSkipsZeroStart(t *testing.T) {
	// Item with no start date/dateTime — should be skipped
	itemJSON := `{"id":"bad","summary":"No Start","start":{},"end":{}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, gcalResponse(itemJSON))
	}))
	defer srv.Close()

	orig := googleCalendarBaseURL
	googleCalendarBaseURL = srv.URL
	defer func() { googleCalendarBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	events, err := svc_calendar(auth).fetchGoogleCalendar(auth.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events (zero start skipped), got %d", len(events))
	}
}

func TestFetchGoogleCalendarWithAttendees(t *testing.T) {
	now := time.Now()
	start := now.Add(time.Hour).UTC().Format(time.RFC3339)
	end := now.Add(2 * time.Hour).UTC().Format(time.RFC3339)
	itemJSON := fmt.Sprintf(
		`{"id":"att","summary":"Meeting","start":{"dateTime":%q},"end":{"dateTime":%q},"attendees":[{"displayName":"Alice","email":"alice@example.com"},{"email":"bob@example.com"}]}`,
		start, end,
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, gcalResponse(itemJSON))
	}))
	defer srv.Close()

	orig := googleCalendarBaseURL
	googleCalendarBaseURL = srv.URL
	defer func() { googleCalendarBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	events, err := svc_calendar(auth).fetchGoogleCalendar(auth.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	if len(events[0].Attendees) != 2 {
		t.Errorf("expected 2 attendees, got %d", len(events[0].Attendees))
	}
	if events[0].Attendees[0] != "Alice" {
		t.Errorf("expected first attendee 'Alice', got %q", events[0].Attendees[0])
	}
	if events[0].Attendees[1] != "bob@example.com" {
		t.Errorf("expected second attendee 'bob@example.com', got %q", events[0].Attendees[1])
	}
}

// svc_calendar creates a CalendarService with the given auth.
func svc_calendar(auth *GoogleAuthService) *CalendarService {
	return NewCalendarServiceWithAuth(config.GoogleConfig{}, auth)
}

func TestCalendarFetchGoogleErrorNoCache(t *testing.T) {
	// Google Calendar returns an error; no cache → falls through to iCal path → empty (no iCal URL)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"error":{"code":500,"message":"Server Error"}}`)
	}))
	defer srv.Close()

	orig := googleCalendarBaseURL
	googleCalendarBaseURL = srv.URL
	defer func() { googleCalendarBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewCalendarServiceWithAuth(config.GoogleConfig{}, auth)
	events, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events (Google error, no iCal URL), got %d", len(events))
	}
}

func TestCalendarFetchGoogleErrorWithCache(t *testing.T) {
	// Google Calendar returns an error; cache is populated → returns cache
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"error":{"code":500,"message":"Server Error"}}`)
	}))
	defer srv.Close()

	orig := googleCalendarBaseURL
	googleCalendarBaseURL = srv.URL
	defer func() { googleCalendarBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewCalendarServiceWithAuth(config.GoogleConfig{}, auth)
	// Populate cache manually
	svc.mu.Lock()
	svc.cache = []models.CalendarEvent{{ID: "cached1", Title: "Cached Event"}}
	svc.mu.Unlock()

	events, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) == 0 {
		t.Error("expected cached events when Google Calendar fails")
	}
}

func TestParseICalEventWithDescription(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(time.Hour)
	end := start.Add(time.Hour)
	dtStart := start.Format("20060102T150405Z")
	dtEnd := end.Format("20060102T150405Z")

	feed := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nUID:desc-test\nSUMMARY:Described Event\nDESCRIPTION:A test description\nDTSTART:" + dtStart + "\nDTEND:" + dtEnd + "\nEND:VEVENT\nEND:VCALENDAR"
	events, err := parseICalTodayEvents(strings.NewReader(feed))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
	if events[0].Description != "A test description" {
		t.Errorf("expected description 'A test description', got %q", events[0].Description)
	}
}

func TestParseICalDateTimeInvalidDate(t *testing.T) {
	params := map[string]string{"VALUE": "DATE"}
	_, ok := parseICalDateTime("not-a-date", params)
	if ok {
		t.Error("expected false for invalid DATE value")
	}
}

func TestParseICalDateTimeInvalidUTC(t *testing.T) {
	params := map[string]string{}
	// Ends with Z but is not valid UTC datetime
	_, ok := parseICalDateTime("BADVALUEZ", params)
	if ok {
		t.Error("expected false for invalid UTC datetime value")
	}
}
