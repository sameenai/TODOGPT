package services

import (
	"bufio"
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

type CalendarService struct {
	cfg   config.GoogleConfig
	cache []models.CalendarEvent
	mu    sync.RWMutex
}

func NewCalendarService(cfg config.GoogleConfig) *CalendarService {
	return &CalendarService{cfg: cfg}
}

// IsLive returns true when an iCal subscription URL is configured.
func (s *CalendarService) IsLive() bool { return s.cfg.ICalURL != "" }

func (s *CalendarService) Fetch() ([]models.CalendarEvent, error) {
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
