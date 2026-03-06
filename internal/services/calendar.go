package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

// googleCalendarBaseURL is a package-level variable so tests can override it.
var googleCalendarBaseURL = "https://www.googleapis.com/calendar/v3"

type CalendarService struct {
	cfg   config.GoogleConfig
	auth  *GoogleAuthService // nil when OAuth is not configured
	cache []models.CalendarEvent
	mu    sync.RWMutex
}

func NewCalendarService(cfg config.GoogleConfig) *CalendarService {
	return &CalendarService{cfg: cfg}
}

// NewCalendarServiceWithAuth creates a CalendarService that can use the Google
// Calendar API when the GoogleAuthService has a valid token.
func NewCalendarServiceWithAuth(cfg config.GoogleConfig, auth *GoogleAuthService) *CalendarService {
	return &CalendarService{cfg: cfg, auth: auth}
}

// IsLive returns true when either Google OAuth is connected or an iCal URL is configured.
func (s *CalendarService) IsLive() bool {
	return (s.auth != nil && s.auth.IsConnected()) || s.cfg.ICalURL != ""
}

func (s *CalendarService) Fetch() ([]models.CalendarEvent, error) {
	// Prefer Google Calendar API when OAuth is connected.
	if s.auth != nil && s.auth.IsConnected() {
		events, err := s.fetchGoogleCalendar(s.auth.Client())
		if err != nil {
			cached := s.GetCached()
			if len(cached) > 0 {
				return cached, nil
			}
			// Fall through to iCal if available
		} else {
			s.mu.Lock()
			s.cache = events
			s.mu.Unlock()
			return events, nil
		}
	}

	if s.cfg.ICalURL == "" {
		return nil, nil
	}
	events, err := s.fetchICalEvents()
	if err != nil {
		cached := s.GetCached()
		if len(cached) > 0 {
			return cached, nil
		}
		return nil, err
	}
	s.mu.Lock()
	s.cache = events
	s.mu.Unlock()
	return events, nil
}

func (s *CalendarService) GetCached() []models.CalendarEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache
}

// ── Google Calendar REST API ───────────────────────────────────────────────────

type gcalEventsResp struct {
	Items []gcalEvent `json:"items"`
	Error *gcalError  `json:"error"`
}

type gcalError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type gcalEvent struct {
	ID          string       `json:"id"`
	Summary     string       `json:"summary"`
	Description string       `json:"description"`
	Location    string       `json:"location"`
	Start       gcalDateTime `json:"start"`
	End         gcalDateTime `json:"end"`
	HangoutLink string       `json:"hangoutLink"`
	Attendees   []gcalAttend `json:"attendees"`
}

type gcalDateTime struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"` // all-day events
	TimeZone string `json:"timeZone"`
}

type gcalAttend struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

func (s *CalendarService) fetchGoogleCalendar(client *http.Client) ([]models.CalendarEvent, error) {
	if client == nil {
		return nil, fmt.Errorf("no authenticated client")
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"%s/calendars/primary/events?timeMin=%s&timeMax=%s&singleEvents=true&orderBy=startTime&maxResults=20",
		googleCalendarBaseURL,
		todayStart.UTC().Format(time.RFC3339),
		todayEnd.UTC().Format(time.RFC3339),
	)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("google calendar request: %w", err)
	}
	defer resp.Body.Close()

	var parsed gcalEventsResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("google calendar decode: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("google calendar API error %d: %s", parsed.Error.Code, parsed.Error.Message)
	}

	var events []models.CalendarEvent
	for _, item := range parsed.Items {
		var startTime, endTime time.Time
		allDay := false

		if item.Start.DateTime != "" {
			startTime, _ = time.Parse(time.RFC3339, item.Start.DateTime)
		} else if item.Start.Date != "" {
			startTime, _ = time.ParseInLocation("2006-01-02", item.Start.Date, now.Location())
			allDay = true
		}
		if item.End.DateTime != "" {
			endTime, _ = time.Parse(time.RFC3339, item.End.DateTime)
		} else if item.End.Date != "" {
			endTime, _ = time.ParseInLocation("2006-01-02", item.End.Date, now.Location())
		}
		if startTime.IsZero() {
			continue
		}

		var attendees []string
		for _, a := range item.Attendees {
			name := a.DisplayName
			if name == "" {
				name = a.Email
			}
			attendees = append(attendees, name)
		}

		events = append(events, models.CalendarEvent{
			ID:          item.ID,
			Title:       item.Summary,
			Description: item.Description,
			Location:    item.Location,
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      allDay,
			MeetingURL:  item.HangoutLink,
			Attendees:   attendees,
			Source:      "google",
		})
	}
	return events, nil
}

func (s *CalendarService) fetchICalEvents() ([]models.CalendarEvent, error) {
	resp, err := http.Get(s.cfg.ICalURL) // #nosec G107 -- user-supplied config value
	if err != nil {
		return nil, fmt.Errorf("fetching iCal URL: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iCal URL returned HTTP %d", resp.StatusCode)
	}
	return parseICalTodayEvents(resp.Body)
}

// parseICalTodayEvents reads an iCalendar stream and returns events that overlap today.
func parseICalTodayEvents(r io.Reader) ([]models.CalendarEvent, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	// Unfold continuation lines (RFC 5545 §3.1: lines folded with CRLF + SP/TAB).
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 128*1024), 128*1024)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && len(lines) > 0 {
			lines[len(lines)-1] += line[1:]
		} else {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var events []models.CalendarEvent
	var cur *models.CalendarEvent

	for _, line := range lines {
		switch line {
		case "BEGIN:VEVENT":
			cur = &models.CalendarEvent{Source: "ical"}
		case "END:VEVENT":
			if cur != nil && !cur.StartTime.IsZero() {
				end := cur.EndTime
				if end.IsZero() {
					end = cur.StartTime.Add(time.Hour)
				}
				if cur.StartTime.Before(todayEnd) && end.After(todayStart) {
					events = append(events, *cur)
				}
			}
			cur = nil
		default:
			if cur == nil {
				continue
			}
			name, value, params := splitICalLine(line)
			switch name {
			case "UID":
				cur.ID = value
			case "SUMMARY":
				cur.Title = unescapeICalText(value)
			case "DESCRIPTION":
				cur.Description = unescapeICalText(value)
			case "LOCATION":
				cur.Location = unescapeICalText(value)
			case "URL":
				cur.MeetingURL = strings.TrimSpace(value)
			case "DTSTART":
				t, allDay := parseICalDateTime(value, params)
				if !t.IsZero() {
					cur.StartTime = t
					cur.AllDay = allDay
				}
			case "DTEND":
				t, _ := parseICalDateTime(value, params)
				if !t.IsZero() {
					cur.EndTime = t
				}
			}
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].StartTime.Before(events[j].StartTime)
	})
	return events, nil
}

// splitICalLine parses "NAME;PARAM=VAL:content" into (NAME, content, params map).
func splitICalLine(line string) (string, string, map[string]string) {
	colon := strings.Index(line, ":")
	if colon < 0 {
		return strings.ToUpper(line), "", nil
	}
	nameAndParams := line[:colon]
	value := line[colon+1:]
	parts := strings.Split(nameAndParams, ";")
	name := strings.ToUpper(parts[0])
	params := make(map[string]string)
	for _, p := range parts[1:] {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			params[strings.ToUpper(kv[0])] = kv[1]
		}
	}
	return name, value, params
}

// parseICalDateTime converts an iCal date/datetime value to time.Time.
// Returns (time, isAllDay).
func parseICalDateTime(value string, params map[string]string) (time.Time, bool) {
	if params["VALUE"] == "DATE" || (len(value) == 8 && !strings.Contains(value, "T")) {
		t, err := time.ParseInLocation("20060102", value, time.Local)
		if err != nil {
			return time.Time{}, false
		}
		return t, true
	}
	if strings.HasSuffix(value, "Z") {
		t, err := time.Parse("20060102T150405Z", value)
		if err != nil {
			return time.Time{}, false
		}
		return t.In(time.Local), false
	}
	loc := time.Local
	if tzid := params["TZID"]; tzid != "" {
		if l, err := time.LoadLocation(tzid); err == nil {
			loc = l
		}
	}
	t, err := time.ParseInLocation("20060102T150405", value, loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, false
}

// unescapeICalText handles iCal text escaping: \n → newline, \, → comma, etc.
func unescapeICalText(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\N`, "\n")
	s = strings.ReplaceAll(s, `\,`, ",")
	s = strings.ReplaceAll(s, `\;`, ";")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
