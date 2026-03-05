package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
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
