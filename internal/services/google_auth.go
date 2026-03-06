package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/todogpt/daily-briefing/internal/config"
)

var googleOAuthScopes = []string{
	"https://www.googleapis.com/auth/calendar.readonly",
	"https://www.googleapis.com/auth/gmail.readonly",
}

// GoogleAuthService manages Google OAuth2 tokens and provides authenticated
// HTTP clients for Calendar and Gmail API calls.
type GoogleAuthService struct {
	oauthCfg     *oauth2.Config
	token        *oauth2.Token
	mu           sync.RWMutex
	tokenPath    string
	pendingState string // CSRF state generated for the current auth flow
	returnTo     string // frontend origin to redirect to after successful auth
}

// NewGoogleAuthService creates the service.  redirectBase should be the
// server's base URL (e.g. "http://localhost:8080") so the callback URL is
// registered correctly with Google.
func NewGoogleAuthService(cfg config.GoogleConfig, redirectBase string, dataDir string) *GoogleAuthService {
	tokenPath := tokenFilePath(dataDir)
	svc := &GoogleAuthService{tokenPath: tokenPath}

	if cfg.ClientID != "" && cfg.ClientSecret != "" {
		svc.oauthCfg = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Scopes:       googleOAuthScopes,
			Endpoint:     google.Endpoint,
			RedirectURL:  redirectBase + "/api/auth/google/callback",
		}
		svc.token = svc.loadToken()
	}

	return svc
}

func tokenFilePath(dataDir string) string {
	if dataDir != "" {
		return filepath.Join(dataDir, "google-token.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".daily-briefing", "google-token.json")
}

// IsConfigured returns true when OAuth2 client credentials are present.
func (g *GoogleAuthService) IsConfigured() bool {
	return g.oauthCfg != nil
}

// IsConnected returns true when a valid token is stored, or when an expired
// token has a refresh token (the oauth2 transport will refresh it automatically).
func (g *GoogleAuthService) IsConnected() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.token == nil {
		return false
	}
	return g.token.Valid() || g.token.RefreshToken != ""
}

// AuthURL generates a Google consent-page URL for the OAuth2 flow and stores
// the CSRF state for later validation.  returnTo is the frontend origin that
// the browser should be redirected to after a successful auth callback.
func (g *GoogleAuthService) AuthURL(returnTo string) string {
	if !g.IsConfigured() {
		return ""
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state := hex.EncodeToString(b)
	g.mu.Lock()
	g.pendingState = state
	g.returnTo = returnTo
	g.mu.Unlock()
	return g.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ReturnTo returns the frontend origin stored during the last AuthURL call.
func (g *GoogleAuthService) ReturnTo() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.returnTo
}

// ValidateState checks the CSRF state returned by Google against the one we sent.
func (g *GoogleAuthService) ValidateState(state string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return state != "" && state == g.pendingState
}

// Exchange trades an authorization code for an access+refresh token pair.
func (g *GoogleAuthService) Exchange(ctx context.Context, code string) error {
	if !g.IsConfigured() {
		return fmt.Errorf("google oauth not configured")
	}
	token, err := g.oauthCfg.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("oauth token exchange: %w", err)
	}
	g.mu.Lock()
	g.token = token
	g.pendingState = "" // consumed
	g.mu.Unlock()
	return g.saveToken(token)
}

// Client returns an *http.Client that automatically refreshes the OAuth token.
// Returns nil when not configured or not connected.
func (g *GoogleAuthService) Client() *http.Client {
	g.mu.RLock()
	token := g.token
	cfg := g.oauthCfg
	g.mu.RUnlock()
	if token == nil || cfg == nil {
		return nil
	}
	return cfg.Client(context.Background(), token)
}

// Disconnect deletes the stored token so the user is logged out.
func (g *GoogleAuthService) Disconnect() error {
	g.mu.Lock()
	g.token = nil
	g.mu.Unlock()
	err := os.Remove(g.tokenPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (g *GoogleAuthService) loadToken() *oauth2.Token {
	data, err := os.ReadFile(g.tokenPath) // #nosec G304 -- known config path
	if err != nil {
		return nil
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil
	}
	return &token
}

func (g *GoogleAuthService) saveToken(token *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(g.tokenPath), 0750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(token, "", "  ") // #nosec G117 -- intentionally serializing OAuth token to user's config dir
	if err != nil {
		return err
	}
	return os.WriteFile(g.tokenPath, data, 0600)
}
