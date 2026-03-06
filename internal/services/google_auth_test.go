package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/todogpt/daily-briefing/internal/config"
)

func testGoogleCfg() config.GoogleConfig {
	return config.GoogleConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
}

func TestGoogleAuthNotConfigured(t *testing.T) {
	svc := NewGoogleAuthService(config.GoogleConfig{}, "http://localhost:8080", "")
	if svc.IsConfigured() {
		t.Error("expected IsConfigured=false when no ClientID/Secret")
	}
	if svc.IsConnected() {
		t.Error("expected IsConnected=false when not configured")
	}
	if url := svc.AuthURL(""); url != "" {
		t.Errorf("expected empty AuthURL, got %q", url)
	}
}

func TestGoogleAuthConfigured(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	if !svc.IsConfigured() {
		t.Error("expected IsConfigured=true when ClientID+Secret set")
	}
	if svc.IsConnected() {
		t.Error("expected IsConnected=false before auth")
	}
}

func TestGoogleAuthURL(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	url := svc.AuthURL("http://localhost:3000")
	if url == "" {
		t.Error("expected non-empty AuthURL")
	}
	// URL should contain accounts.google.com
	if len(url) < 10 {
		t.Errorf("URL too short: %q", url)
	}
}

func TestGoogleAuthStateValidation(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)

	// Before any AuthURL call, state is empty → validation fails
	if svc.ValidateState("anything") {
		t.Error("expected ValidateState=false before AuthURL")
	}
	if svc.ValidateState("") {
		t.Error("expected ValidateState=false for empty state")
	}

	// After AuthURL, state matches
	svc.AuthURL("http://localhost:3000")
	svc.mu.RLock()
	state := svc.pendingState
	svc.mu.RUnlock()

	if !svc.ValidateState(state) {
		t.Error("expected ValidateState=true for correct state")
	}
	if svc.ValidateState("wrong-state") {
		t.Error("expected ValidateState=false for wrong state")
	}
}

func TestGoogleAuthReturnTo(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)

	if svc.ReturnTo() != "" {
		t.Error("expected empty ReturnTo before AuthURL")
	}

	svc.AuthURL("http://localhost:3000")
	if svc.ReturnTo() != "http://localhost:3000" {
		t.Errorf("expected ReturnTo=http://localhost:3000, got %q", svc.ReturnTo())
	}
}

func TestGoogleAuthClientNilWhenNotConnected(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	if svc.Client() != nil {
		t.Error("expected nil client when not connected")
	}
}

func TestGoogleAuthClientNilWhenNotConfigured(t *testing.T) {
	svc := NewGoogleAuthService(config.GoogleConfig{}, "http://localhost:8080", "")
	if svc.Client() != nil {
		t.Error("expected nil client when not configured")
	}
}

func TestGoogleAuthTokenSaveLoad(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)

	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
		TokenType:    "Bearer",
	}

	if err := svc.saveToken(token); err != nil {
		t.Fatalf("saveToken error: %v", err)
	}

	loaded := svc.loadToken()
	if loaded == nil {
		t.Fatal("expected loaded token")
	}
	if loaded.AccessToken != token.AccessToken {
		t.Errorf("expected AccessToken %q, got %q", token.AccessToken, loaded.AccessToken)
	}
}

func TestGoogleAuthLoadTokenInvalidFile(t *testing.T) {
	dir := t.TempDir()
	// Write invalid JSON to token file
	tokenPath := filepath.Join(dir, "google-token.json")
	os.WriteFile(tokenPath, []byte("not-json"), 0600) //nolint:errcheck

	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	// loadToken is called in constructor; should return nil on bad JSON
	if svc.token != nil {
		t.Error("expected nil token for invalid JSON")
	}
}

func TestGoogleAuthLoadTokenMissingFile(t *testing.T) {
	dir := t.TempDir()
	// No token file
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	if svc.token != nil {
		t.Error("expected nil token when file missing")
	}
}

func TestGoogleAuthDisconnect(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)

	// Save a token first
	token := &oauth2.Token{
		AccessToken: "tok",
		Expiry:      time.Now().Add(time.Hour),
		TokenType:   "Bearer",
	}
	svc.saveToken(token) //nolint:errcheck
	svc.mu.Lock()
	svc.token = token
	svc.mu.Unlock()

	if !svc.IsConnected() {
		t.Fatal("expected IsConnected=true before disconnect")
	}

	if err := svc.Disconnect(); err != nil {
		t.Fatalf("Disconnect error: %v", err)
	}

	if svc.IsConnected() {
		t.Error("expected IsConnected=false after disconnect")
	}

	// File should be removed
	if _, err := os.Stat(filepath.Join(dir, "google-token.json")); !os.IsNotExist(err) {
		t.Error("expected token file to be deleted after disconnect")
	}
}

func TestGoogleAuthDisconnectNoFile(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	// Disconnect without ever saving a token — should not error
	if err := svc.Disconnect(); err != nil {
		t.Errorf("unexpected error disconnecting with no token file: %v", err)
	}
}

func TestGoogleAuthIsConnectedExpiredToken(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)

	// Inject expired token
	svc.mu.Lock()
	svc.token = &oauth2.Token{
		AccessToken: "expired",
		Expiry:      time.Now().Add(-time.Hour), // expired
		TokenType:   "Bearer",
	}
	svc.mu.Unlock()

	// Without refresh token, expired token is not valid
	if svc.IsConnected() {
		t.Error("expected IsConnected=false for expired token without refresh token")
	}
}

func TestGoogleAuthIsConnectedWithRefreshToken(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)

	// Token with refresh token is considered valid even if AccessToken is expired
	svc.mu.Lock()
	svc.token = &oauth2.Token{
		AccessToken:  "expired",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(-time.Hour),
		TokenType:    "Bearer",
	}
	svc.mu.Unlock()

	// oauth2.Token.Valid() returns true if there's a refresh token
	if !svc.IsConnected() {
		t.Error("expected IsConnected=true when refresh token is present")
	}
}

func TestGoogleAuthExchangeNotConfigured(t *testing.T) {
	svc := NewGoogleAuthService(config.GoogleConfig{}, "http://localhost:8080", "")
	if err := svc.Exchange(context.Background(), "some-code"); err == nil {
		t.Error("expected error when not configured")
	}
}

func TestGoogleAuthExchangeServerError(t *testing.T) {
	// Serve a mock token endpoint that returns an error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"}) //nolint:errcheck
	}))
	defer ts.Close()

	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	// Override the oauth2 endpoint to point at our mock server
	svc.oauthCfg.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	err := svc.Exchange(context.Background(), "bad-code")
	if err == nil {
		t.Error("expected error on bad token exchange")
	}
}

func TestTokenFilePath(t *testing.T) {
	path := tokenFilePath("/tmp/testdata")
	if path != "/tmp/testdata/google-token.json" {
		t.Errorf("unexpected path: %q", path)
	}

	// Empty dataDir uses home dir
	path = tokenFilePath("")
	if path == "" {
		t.Error("expected non-empty path for empty dataDir")
	}
}

// validToken returns an oauth2.Token that is currently valid.
func validToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken: "test-access",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}
}

func TestGoogleAuthExchangeSuccess(t *testing.T) {
	// Mock token endpoint returning a valid token
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"access_token":"ya29.test",
			"token_type":"Bearer",
			"expires_in":3600,
			"refresh_token":"1//test"
		}`)
	}))
	defer ts.Close()

	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	svc.oauthCfg.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	if err := svc.Exchange(context.Background(), "valid-code"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !svc.IsConnected() {
		t.Error("expected IsConnected=true after successful exchange")
	}
}

func TestGoogleAuthSaveTokenMkdirError(t *testing.T) {
	dir := t.TempDir()
	svc := NewGoogleAuthService(testGoogleCfg(), "http://localhost:8080", dir)
	// Point tokenPath to an impossible location
	svc.tokenPath = "/dev/null/sub/token.json"

	token := validToken()
	err := svc.saveToken(token)
	if err == nil {
		t.Error("expected error when MkdirAll fails")
	}
}
