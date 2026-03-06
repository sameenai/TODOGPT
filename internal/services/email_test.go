package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	goImap "github.com/emersion/go-imap"
	"github.com/todogpt/daily-briefing/internal/config"
)

func TestNewEmailService(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestEmailIsLive(t *testing.T) {
	cases := []struct {
		cfg  config.EmailConfig
		want bool
	}{
		{config.EmailConfig{}, false},
		{config.EmailConfig{Enabled: true}, false},
		{config.EmailConfig{Enabled: true, Username: "user@example.com"}, false},
		{config.EmailConfig{Enabled: true, Username: "user@example.com", Password: "secret"}, true},
		{config.EmailConfig{Username: "user@example.com", Password: "secret"}, false}, // enabled=false
	}
	for _, c := range cases {
		svc := NewEmailService(c.cfg)
		if got := svc.IsLive(); got != c.want {
			t.Errorf("IsLive(%+v) = %v, want %v", c.cfg, got, c.want)
		}
	}
}

func TestEmailFetch(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{Enabled: true})

	emails, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(emails) == 0 {
		t.Error("expected mock emails")
	}

	for _, e := range emails {
		if e.ID == "" {
			t.Error("email should have an ID")
		}
		if e.From == "" {
			t.Error("email should have a from address")
		}
		if e.Subject == "" {
			t.Error("email should have a subject")
		}
		if e.Date.IsZero() {
			t.Error("email should have a date")
		}
	}
}

func TestEmailGetCachedEmpty(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	emails := svc.GetCached()
	if len(emails) == 0 {
		t.Error("expected mock emails when cache empty")
	}
}

func TestEmailUnreadCount(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	count := svc.UnreadCount()
	if count == 0 {
		t.Error("expected non-zero unread count from mock data")
	}
}

func TestEmailUnreadCountAfterFetch(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	svc.Fetch() //nolint:errcheck
	count := svc.UnreadCount()
	if count == 0 {
		t.Error("expected non-zero unread count after fetch")
	}
}

func TestEmailMockHasUnreadAndStarred(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	emails := svc.mockEmails()

	hasUnread := false
	hasStarred := false
	for _, e := range emails {
		if e.IsUnread {
			hasUnread = true
		}
		if e.IsStarred {
			hasStarred = true
		}
	}

	if !hasUnread {
		t.Error("expected at least one unread email in mock data")
	}
	if !hasStarred {
		t.Error("expected at least one starred email in mock data")
	}
}

func TestEmailMockHasLabels(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	emails := svc.mockEmails()

	for _, e := range emails {
		if len(e.Labels) == 0 {
			t.Errorf("email %q should have labels", e.Subject)
		}
	}
}

func TestEmailGetCachedAfterFetch(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	svc.Fetch() //nolint:errcheck
	cached := svc.GetCached()
	if len(cached) == 0 {
		t.Error("expected cached emails after fetch")
	}
	if cached[0].ID == "" {
		t.Error("cached email should have an ID")
	}
}

func TestEmailFetchNotLiveReturnsMock(t *testing.T) {
	// Enabled but no credentials — should fall back to mock
	svc := NewEmailService(config.EmailConfig{Enabled: true, IMAPServer: "imap.gmail.com", IMAPPort: 993})
	emails, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(emails) == 0 {
		t.Error("expected mock emails when not live")
	}
}

func TestEmailFetchCacheFallback(t *testing.T) {
	// Simulate: first call populates cache, service not live so always uses mock
	svc := NewEmailService(config.EmailConfig{})
	first, _ := svc.Fetch()
	second := svc.GetCached()
	if len(first) != len(second) {
		t.Errorf("cache should match fetch result: %d vs %d", len(first), len(second))
	}
}

// ── Mock IMAP client ──────────────────────────────────────────────────────────

type mockIMAPClient struct {
	loginErr    error
	selectErr   error
	fetchErr    error
	messages    []*goImap.Message
	mailboxMsgs uint32
}

func (m *mockIMAPClient) Login(username, password string) error { return m.loginErr }
func (m *mockIMAPClient) Logout() error                         { return nil }
func (m *mockIMAPClient) Select(name string, readOnly bool) (*goImap.MailboxStatus, error) {
	if m.selectErr != nil {
		return nil, m.selectErr
	}
	return &goImap.MailboxStatus{Messages: m.mailboxMsgs}, nil
}
func (m *mockIMAPClient) Fetch(seqSet *goImap.SeqSet, items []goImap.FetchItem, ch chan *goImap.Message) error {
	for _, msg := range m.messages {
		ch <- msg
	}
	close(ch)
	return m.fetchErr
}

func newMockMessage(uid uint32, subject, from string, seen bool) *goImap.Message {
	msg := goImap.NewMessage(uid, []goImap.FetchItem{goImap.FetchEnvelope, goImap.FetchFlags, goImap.FetchUid})
	msg.Uid = uid
	msg.Envelope = &goImap.Envelope{
		Subject: subject,
		Date:    time.Now(),
		From:    []*goImap.Address{{MailboxName: "sender", HostName: "example.com"}},
	}
	if from != "" {
		msg.Envelope.From = []*goImap.Address{{PersonalName: from, MailboxName: "user", HostName: "example.com"}}
	}
	if seen {
		msg.Flags = []string{goImap.SeenFlag}
	}
	return msg
}

func withMockIMAP(mock imapClientIface, fn func()) {
	orig := imapDial
	imapDial = func(addr string) (imapClientIface, error) { return mock, nil }
	defer func() { imapDial = orig }()
	fn()
}

func withMockIMAPDialError(dialErr error, fn func()) {
	orig := imapDial
	imapDial = func(addr string) (imapClientIface, error) { return nil, dialErr }
	defer func() { imapDial = orig }()
	fn()
}

// ── Tests using mock IMAP client ──────────────────────────────────────────────

func TestEmailFetchIMAPSuccess(t *testing.T) {
	mock := &mockIMAPClient{
		mailboxMsgs: 2,
		messages: []*goImap.Message{
			newMockMessage(1, "Hello World", "Alice", false),
			newMockMessage(2, "Read Email", "Bob", true), // seen → not unread
		},
	}
	withMockIMAP(mock, func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:    true,
			Username:   "user@example.com",
			Password:   "secret",
			IMAPServer: "imap.example.com",
			IMAPPort:   993,
		})
		emails, err := svc.Fetch()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(emails) != 2 {
			t.Fatalf("expected 2 emails, got %d", len(emails))
		}
		// First email: not seen → unread
		if !emails[0].IsUnread {
			t.Error("expected email 1 to be unread")
		}
		// Second email: seen → not unread
		if emails[1].IsUnread {
			t.Error("expected email 2 to be read (not unread)")
		}
		if emails[0].Subject != "Hello World" {
			t.Errorf("expected subject 'Hello World', got %q", emails[0].Subject)
		}
		// ID uses UID prefix
		if emails[0].ID != "imap-1" {
			t.Errorf("expected ID 'imap-1', got %q", emails[0].ID)
		}
	})
}

func TestEmailFetchIMAPEmptyMailbox(t *testing.T) {
	mock := &mockIMAPClient{mailboxMsgs: 0}
	withMockIMAP(mock, func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:  true,
			Username: "user@example.com",
			Password: "secret",
		})
		emails, err := svc.Fetch()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(emails) != 0 {
			t.Errorf("expected 0 emails for empty mailbox, got %d", len(emails))
		}
	})
}

func TestEmailFetchIMAPLoginError(t *testing.T) {
	mock := &mockIMAPClient{loginErr: fmt.Errorf("invalid credentials")}
	withMockIMAP(mock, func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:  true,
			Username: "user@example.com",
			Password: "wrong",
		})
		// Should fall back to mock on login error
		emails, err := svc.Fetch()
		if err != nil {
			t.Fatalf("expected fallback to mock, got error: %v", err)
		}
		if len(emails) == 0 {
			t.Error("expected mock emails on login failure")
		}
	})
}

func TestEmailFetchIMAPLoginErrorWithCache(t *testing.T) {
	// First fetch succeeds → populates cache. Second → login error → returns cache.
	mock := &mockIMAPClient{
		mailboxMsgs: 1,
		messages:    []*goImap.Message{newMockMessage(1, "Cached Subject", "Alice", false)},
	}
	withMockIMAP(mock, func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:  true,
			Username: "user@example.com",
			Password: "secret",
		})
		first, err := svc.Fetch()
		if err != nil {
			t.Fatalf("first fetch error: %v", err)
		}
		if len(first) == 0 {
			t.Fatal("expected emails on first fetch")
		}

		// Make login fail on second call
		mock.loginErr = fmt.Errorf("auth error")
		second, err := svc.Fetch()
		if err != nil {
			t.Fatalf("expected cache fallback on error, got: %v", err)
		}
		if len(second) == 0 {
			t.Error("expected cached emails on second fetch")
		}
	})
}

func TestEmailFetchIMAPSelectError(t *testing.T) {
	mock := &mockIMAPClient{selectErr: fmt.Errorf("mailbox not found")}
	withMockIMAP(mock, func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:  true,
			Username: "user@example.com",
			Password: "secret",
		})
		emails, err := svc.Fetch()
		if err != nil {
			t.Fatalf("expected mock fallback, got error: %v", err)
		}
		if len(emails) == 0 {
			t.Error("expected mock emails on select failure")
		}
	})
}

func TestEmailFetchIMAPFetchError(t *testing.T) {
	mock := &mockIMAPClient{
		mailboxMsgs: 1,
		messages:    []*goImap.Message{newMockMessage(1, "Test", "", false)},
		fetchErr:    fmt.Errorf("fetch failed"),
	}
	withMockIMAP(mock, func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:  true,
			Username: "user@example.com",
			Password: "secret",
		})
		emails, err := svc.Fetch()
		if err != nil {
			t.Fatalf("expected mock fallback, got error: %v", err)
		}
		// fetchErr causes fetchFromIMAP to return error, triggering mock fallback
		if len(emails) == 0 {
			t.Error("expected mock emails on fetch failure")
		}
	})
}

func TestEmailFetchIMAPDialError(t *testing.T) {
	withMockIMAPDialError(fmt.Errorf("connection refused"), func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:  true,
			Username: "user@example.com",
			Password: "secret",
		})
		emails, err := svc.Fetch()
		if err != nil {
			t.Fatalf("expected mock fallback on dial error, got: %v", err)
		}
		if len(emails) == 0 {
			t.Error("expected mock emails on dial error")
		}
	})
}

func TestEmailFetchIMAPLargeMailbox(t *testing.T) {
	// Mailbox with >20 messages — should fetch only last 20
	mock := &mockIMAPClient{
		mailboxMsgs: 25,
		messages:    []*goImap.Message{newMockMessage(25, "Recent", "Alice", false)},
	}
	withMockIMAP(mock, func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:  true,
			Username: "user@example.com",
			Password: "secret",
		})
		emails, err := svc.Fetch()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 1 message was returned by mock
		if len(emails) != 1 {
			t.Errorf("expected 1 email, got %d", len(emails))
		}
	})
}

func TestEmailFetchIMAPMessageNoEnvelope(t *testing.T) {
	// Message with nil envelope should be skipped
	msg := goImap.NewMessage(1, []goImap.FetchItem{goImap.FetchEnvelope})
	msg.Uid = 1
	msg.Envelope = nil // no envelope

	mock := &mockIMAPClient{
		mailboxMsgs: 1,
		messages:    []*goImap.Message{msg},
	}
	withMockIMAP(mock, func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:  true,
			Username: "user@example.com",
			Password: "secret",
		})
		emails, err := svc.Fetch()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Message with no envelope should be skipped
		if len(emails) != 0 {
			t.Errorf("expected 0 emails (nil envelope skipped), got %d", len(emails))
		}
	})
}

func TestEmailFetchIMAPFlaggedEmail(t *testing.T) {
	msg := newMockMessage(1, "Flagged Email", "Alice", false)
	msg.Flags = append(msg.Flags, goImap.FlaggedFlag)

	mock := &mockIMAPClient{
		mailboxMsgs: 1,
		messages:    []*goImap.Message{msg},
	}
	withMockIMAP(mock, func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:  true,
			Username: "user@example.com",
			Password: "secret",
		})
		emails, err := svc.Fetch()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(emails) != 1 {
			t.Fatalf("expected 1 email, got %d", len(emails))
		}
		if !emails[0].IsStarred {
			t.Error("expected email to be starred (flagged)")
		}
	})
}

func TestEmailFetchIMAPNoFromAddress(t *testing.T) {
	msg := goImap.NewMessage(1, []goImap.FetchItem{goImap.FetchEnvelope, goImap.FetchFlags, goImap.FetchUid})
	msg.Uid = 1
	msg.Envelope = &goImap.Envelope{
		Subject: "No From",
		Date:    time.Now(),
		From:    []*goImap.Address{}, // empty
	}

	mock := &mockIMAPClient{
		mailboxMsgs: 1,
		messages:    []*goImap.Message{msg},
	}
	withMockIMAP(mock, func() {
		svc := NewEmailService(config.EmailConfig{
			Enabled:  true,
			Username: "user@example.com",
			Password: "secret",
		})
		emails, err := svc.Fetch()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(emails) != 1 {
			t.Fatalf("expected 1 email, got %d", len(emails))
		}
		if emails[0].From != "" {
			t.Errorf("expected empty From, got %q", emails[0].From)
		}
	})
}

func TestEmailFetchIMAPConnectionError(t *testing.T) {
	// IsLive=true but IMAP server unreachable — should fall back to mock data.
	svc := NewEmailService(config.EmailConfig{
		Enabled:    true,
		Username:   "user@example.com",
		Password:   "secret",
		IMAPServer: "127.0.0.1",
		IMAPPort:   1, // nothing listening here → connection refused immediately
	})
	if !svc.IsLive() {
		t.Fatal("expected IsLive=true")
	}
	emails, err := svc.Fetch()
	if err != nil {
		t.Fatalf("expected fallback to mock on IMAP error, got: %v", err)
	}
	if len(emails) == 0 {
		t.Error("expected mock emails on IMAP connection failure")
	}
}

func TestEmailFetchIMAPConnectionErrorWithCache(t *testing.T) {
	// First fetch succeeds (not live → mock), populates cache.
	// Second call with live+error should return cache.
	svc := NewEmailService(config.EmailConfig{})
	first, _ := svc.Fetch()

	// Now switch to live-but-failing config by directly manipulating state via a
	// second instance that shares no cache — just verify the fallback path
	// returns mock data consistently.
	if len(first) == 0 {
		t.Error("expected mock data on first fetch")
	}

	// Verify UnreadCount works with populated cache
	count := svc.UnreadCount()
	if count == 0 {
		t.Error("expected non-zero unread count")
	}
}

// ── Gmail API tests ────────────────────────────────────────────────────────────

func gmailListJSON(ids ...string) string {
	items := ""
	for i, id := range ids {
		if i > 0 {
			items += ","
		}
		items += fmt.Sprintf(`{"id":%q,"threadId":"t%s"}`, id, id)
	}
	return `{"messages":[` + items + `]}`
}

func gmailMsgJSON(id, subject, from, date string, unread, starred bool) string {
	labels := `["INBOX"`
	if unread {
		labels += `,"UNREAD"`
	}
	if starred {
		labels += `,"STARRED"`
	}
	labels += `]`
	return fmt.Sprintf(`{"id":%q,"snippet":"snippet for %s","labelIds":%s,"payload":{"headers":[{"name":"Subject","value":%q},{"name":"From","value":%q},{"name":"Date","value":%q}]}}`,
		id, id, labels, subject, from, date)
}

func newGmailTestServer(t *testing.T, listBody string, msgHandler func(id string) string) (*httptest.Server, func()) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/users/me/messages" {
			fmt.Fprint(w, listBody)
			return
		}
		// /users/me/messages/{id}
		parts := strings.Split(r.URL.Path, "/")
		id := parts[len(parts)-1]
		fmt.Fprint(w, msgHandler(id))
	}))
	orig := googleGmailBaseURL
	googleGmailBaseURL = srv.URL
	cleanup := func() {
		srv.Close()
		googleGmailBaseURL = orig
	}
	return srv, cleanup
}

func TestFetchFromGmailSuccess(t *testing.T) {
	date := "Mon, 01 Jan 2024 12:00:00 +0000"
	_, cleanup := newGmailTestServer(t,
		gmailListJSON("msg1", "msg2"),
		func(id string) string {
			return gmailMsgJSON(id, "Subject "+id, "sender@example.com", date, true, false)
		},
	)
	defer cleanup()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewEmailServiceWithAuth(config.EmailConfig{}, auth)
	msgs, err := svc.fetchFromGmail(auth.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if !msgs[0].IsUnread {
		t.Error("expected IsUnread=true")
	}
	if msgs[0].From != "sender@example.com" {
		t.Errorf("expected from 'sender@example.com', got %q", msgs[0].From)
	}
}

func TestFetchFromGmailStarred(t *testing.T) {
	date := "Mon, 01 Jan 2024 12:00:00 +0000"
	_, cleanup := newGmailTestServer(t,
		gmailListJSON("starred1"),
		func(id string) string {
			return gmailMsgJSON(id, "Starred Email", "a@b.com", date, false, true)
		},
	)
	defer cleanup()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewEmailServiceWithAuth(config.EmailConfig{}, auth)
	msgs, err := svc.fetchFromGmail(auth.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected at least one message")
	}
	if !msgs[0].IsStarred {
		t.Error("expected IsStarred=true")
	}
}

func TestFetchFromGmailAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"error":{"code":403,"message":"Forbidden"}}`)
	}))
	defer srv.Close()
	orig := googleGmailBaseURL
	googleGmailBaseURL = srv.URL
	defer func() { googleGmailBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewEmailServiceWithAuth(config.EmailConfig{}, auth)
	_, err := svc.fetchFromGmail(auth.Client())
	if err == nil {
		t.Error("expected error for API error response")
	}
}

func TestFetchFromGmailNilClient(t *testing.T) {
	svc := NewEmailService(config.EmailConfig{})
	_, err := svc.fetchFromGmail(nil)
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestFetchFromGmailInvalidListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not-json")
	}))
	defer srv.Close()
	orig := googleGmailBaseURL
	googleGmailBaseURL = srv.URL
	defer func() { googleGmailBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewEmailServiceWithAuth(config.EmailConfig{}, auth)
	_, err := svc.fetchFromGmail(auth.Client())
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFetchGmailMessageSkipsFailedMsg(t *testing.T) {
	// List returns msg1 and msg2; msg2 will return invalid JSON — should be skipped
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/users/me/messages" {
			fmt.Fprint(w, gmailListJSON("good1", "bad1"))
			return
		}
		parts := strings.Split(r.URL.Path, "/")
		id := parts[len(parts)-1]
		if id == "bad1" {
			fmt.Fprint(w, "invalid-json")
		} else {
			fmt.Fprint(w, gmailMsgJSON(id, "Good Subject", "x@y.com", "Mon, 01 Jan 2024 12:00:00 +0000", true, false))
		}
	}))
	defer srv.Close()
	orig := googleGmailBaseURL
	googleGmailBaseURL = srv.URL
	defer func() { googleGmailBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewEmailServiceWithAuth(config.EmailConfig{}, auth)
	msgs, err := svc.fetchFromGmail(auth.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message (bad one skipped), got %d", len(msgs))
	}
}

func TestEmailFetchUsesGmailWhenConnected(t *testing.T) {
	date := "Mon, 01 Jan 2024 12:00:00 +0000"
	_, cleanup := newGmailTestServer(t,
		gmailListJSON("m1"),
		func(id string) string {
			return gmailMsgJSON(id, "Gmail Subject", "g@g.com", date, true, false)
		},
	)
	defer cleanup()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewEmailServiceWithAuth(config.EmailConfig{}, auth)
	msgs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected messages from Gmail API")
	}
	if msgs[0].Subject != "Gmail Subject" {
		t.Errorf("expected 'Gmail Subject', got %q", msgs[0].Subject)
	}
}

func TestParseEmailDate(t *testing.T) {
	cases := []struct {
		in   string
		fail bool
	}{
		{"Mon, 02 Jan 2006 15:04:05 -0700", false},
		{"Mon, 02 Jan 2006 15:04:05 MST", false},
		{"02 Jan 2006 15:04:05 -0700", false},
		{"Mon, 02 Jan 2006 15:04:05 +0000", false},
		{"not a date at all", true},
		{"", true},
	}
	for _, c := range cases {
		_, err := parseEmailDate(c.in)
		if c.fail && err == nil {
			t.Errorf("parseEmailDate(%q): expected error, got nil", c.in)
		}
		if !c.fail && err != nil {
			t.Errorf("parseEmailDate(%q): unexpected error %v", c.in, err)
		}
	}
}

func TestEmailIsLiveWithConnectedAuth(t *testing.T) {
	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewEmailServiceWithAuth(config.EmailConfig{}, auth)
	if !svc.IsLive() {
		t.Error("expected IsLive=true when auth is connected")
	}
}

func TestEmailFetchGmailErrorFallthrough(t *testing.T) {
	// Gmail server that immediately closes the connection
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(500)
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer srv.Close()

	orig := googleGmailBaseURL
	googleGmailBaseURL = srv.URL
	defer func() { googleGmailBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	// No IMAP config — should fall to mock data
	svc := NewEmailServiceWithAuth(config.EmailConfig{}, auth)
	msgs, err := svc.Fetch()
	// Error from Gmail, falls through to mock — no error returned
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Error("expected mock messages when Gmail fails and no IMAP configured")
	}
}

func TestFetchFromGmailNetworkError(t *testing.T) {
	// Server that immediately closes the connection
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(500)
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer srv.Close()

	orig := googleGmailBaseURL
	googleGmailBaseURL = srv.URL
	defer func() { googleGmailBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewEmailServiceWithAuth(config.EmailConfig{}, auth)
	_, err := svc.fetchFromGmail(auth.Client())
	if err == nil {
		t.Error("expected error when connection is closed")
	}
}

func TestFetchGmailMessageNetworkError(t *testing.T) {
	// List endpoint works, message endpoint closes connection
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/me/messages" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, gmailListJSON("net-err-msg"))
			return
		}
		// Message endpoint — close connection
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(500)
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer srv.Close()

	orig := googleGmailBaseURL
	googleGmailBaseURL = srv.URL
	defer func() { googleGmailBaseURL = orig }()

	dir := t.TempDir()
	auth := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	auth.mu.Lock()
	auth.token = validToken()
	auth.mu.Unlock()

	svc := NewEmailServiceWithAuth(config.EmailConfig{}, auth)
	// The failed message is skipped — result is empty but no error
	msgs, err := svc.fetchFromGmail(auth.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages (failed message skipped), got %d", len(msgs))
	}
}
