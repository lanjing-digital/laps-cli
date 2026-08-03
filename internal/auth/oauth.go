package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const ClientID = "laps-cli"

type BrowserOpener func(string) error

type LoginOptions struct {
	BaseURL     string
	Store       Store
	HTTPClient  *http.Client
	OpenBrowser BrowserOpener
	NoBrowser   bool
	Timeout     time.Duration
	Output      io.Writer
}

type tokenResponse struct {
	AccessToken           string `json:"access_token"`
	TokenType             string `json:"token_type"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	Scope                 string `json:"scope"`
	User                  User   `json:"user"`
}

type callbackResult struct {
	code string
	err  error
}

func Login(ctx context.Context, options LoginOptions) (Credentials, error) {
	baseURL, err := normalizeBaseURL(options.BaseURL)
	if err != nil {
		return Credentials{}, err
	}
	if options.Store == nil {
		return Credentials{}, errors.New("credentials store is required")
	}
	if options.Timeout <= 0 {
		options.Timeout = 5 * time.Minute
	}
	if options.Output == nil {
		options.Output = io.Discard
	}
	if options.OpenBrowser == nil {
		options.OpenBrowser = OpenBrowser
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return Credentials{}, fmt.Errorf("start OAuth callback listener: %w", err)
	}
	defer listener.Close()

	state, err := randomValue(32)
	if err != nil {
		return Credentials{}, err
	}
	verifier, err := randomValue(64)
	if err != nil {
		return Credentials{}, err
	}
	challengeBytes := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])
	redirectURI := "http://" + listener.Addr().String() + "/oauth/callback"
	authorizeURL := baseURL + "/api/auth/oauth/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {ClientID},
		"redirect_uri":          {redirectURI},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}.Encode()

	resultChannel := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", func(response http.ResponseWriter, request *http.Request) {
		if subtle.ConstantTimeCompare([]byte(request.URL.Query().Get("state")), []byte(state)) != 1 {
			http.Error(response, "OAuth state mismatch", http.StatusBadRequest)
			select {
			case resultChannel <- callbackResult{err: errors.New("OAuth state mismatch")}:
			default:
			}
			return
		}
		if oauthError := request.URL.Query().Get("error"); oauthError != "" {
			description := request.URL.Query().Get("error_description")
			http.Error(response, oauthError, http.StatusBadRequest)
			select {
			case resultChannel <- callbackResult{err: fmt.Errorf("OAuth authorization failed: %s %s", oauthError, description)}:
			default:
			}
			return
		}
		code := request.URL.Query().Get("code")
		if code == "" {
			http.Error(response, "Missing authorization code", http.StatusBadRequest)
			select {
			case resultChannel <- callbackResult{err: errors.New("OAuth callback did not include a code")}:
			default:
			}
			return
		}
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(response, "<!doctype html><meta charset=utf-8><title>LAPS CLI</title><p>Authorization complete. You can close this window.</p>")
		select {
		case resultChannel <- callbackResult{code: code}:
		default:
		}
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())

	fmt.Fprintf(options.Output, "Open this URL to authorize laps-cli:\n%s\n", authorizeURL)
	if !options.NoBrowser {
		if err := options.OpenBrowser(authorizeURL); err != nil {
			fmt.Fprintf(options.Output, "Could not open the browser automatically: %v\n", err)
		}
	}

	waitContext, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	var callback callbackResult
	select {
	case callback = <-resultChannel:
	case <-waitContext.Done():
		return Credentials{}, fmt.Errorf("OAuth login timed out: %w", waitContext.Err())
	}
	if callback.err != nil {
		return Credentials{}, callback.err
	}

	tokens, err := requestToken(waitContext, options.HTTPClient, baseURL, url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {ClientID},
		"code":          {callback.code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	})
	if err != nil {
		return Credentials{}, err
	}
	credentials := credentialsFromResponse(baseURL, tokens, time.Now())
	if err := options.Store.Save(credentials); err != nil {
		return Credentials{}, err
	}
	return credentials, nil
}

type Manager struct {
	BaseURL     string
	Store       Store
	HTTPClient  *http.Client
	RefreshSkew time.Duration

	mu          sync.Mutex
	credentials *Credentials
}

func (m *Manager) AccessToken(ctx context.Context, forceRefresh bool) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	credentials, err := m.load()
	if err != nil {
		return "", fmt.Errorf("OAuth login required; run `laps-cli auth login`: %w", err)
	}
	baseURL, err := normalizeBaseURL(m.BaseURL)
	if err != nil {
		return "", err
	}
	if credentials.BaseURL != baseURL {
		return "", fmt.Errorf("credentials belong to %s; run `laps-cli auth login --base-url %s`", credentials.BaseURL, baseURL)
	}
	skew := m.RefreshSkew
	if skew <= 0 {
		skew = 60 * time.Second
	}
	if !forceRefresh && time.Now().Add(skew).Before(credentials.ExpiresAt) {
		return credentials.AccessToken, nil
	}
	if time.Now().After(credentials.RefreshTokenExpiresAt) {
		return "", errors.New("OAuth refresh token expired; run `laps-cli auth login` again")
	}

	tokens, err := requestToken(ctx, m.HTTPClient, baseURL, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {ClientID},
		"refresh_token": {credentials.RefreshToken},
	})
	if err != nil {
		return "", fmt.Errorf("refresh OAuth token: %w", err)
	}
	updated := credentialsFromResponse(baseURL, tokens, time.Now())
	if err := m.Store.Save(updated); err != nil {
		return "", err
	}
	m.credentials = &updated
	return updated.AccessToken, nil
}

func (m *Manager) Credentials() (Credentials, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.load()
}

func (m *Manager) Logout(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	credentials, err := m.load()
	if err != nil && !errors.Is(err, ErrNotLoggedIn) {
		return err
	}
	if err == nil {
		baseURL, normalizeErr := normalizeBaseURL(credentials.BaseURL)
		if normalizeErr != nil {
			return normalizeErr
		}
		form := url.Values{"token": {credentials.RefreshToken}, "client_id": {ClientID}}
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/auth/oauth/revoke", strings.NewReader(form.Encode()))
		if requestErr != nil {
			return requestErr
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response, callErr := httpClient(m.HTTPClient).Do(request)
		if callErr != nil {
			return fmt.Errorf("revoke OAuth token: %w", callErr)
		}
		raw, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read OAuth revoke response: %w", readErr)
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Errorf("revoke OAuth token: HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(raw)))
		}
	}
	m.credentials = nil
	return m.Store.Remove()
}

func (m *Manager) load() (Credentials, error) {
	if m.credentials != nil {
		return *m.credentials, nil
	}
	credentials, err := m.Store.Load()
	if err != nil {
		return Credentials{}, err
	}
	m.credentials = &credentials
	return credentials, nil
}

func requestToken(ctx context.Context, configuredClient *http.Client, baseURL string, form url.Values) (tokenResponse, error) {
	var tokens tokenResponse
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/auth/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return tokens, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	response, err := httpClient(configuredClient).Do(request)
	if err != nil {
		return tokens, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return tokens, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return tokens, fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(raw)))
	}
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return tokens, fmt.Errorf("parse token response: %w", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" || !strings.EqualFold(tokens.TokenType, "Bearer") {
		return tokens, errors.New("token response is incomplete")
	}
	return tokens, nil
}

func credentialsFromResponse(baseURL string, tokens tokenResponse, now time.Time) Credentials {
	return Credentials{
		BaseURL:               baseURL,
		AccessToken:           tokens.AccessToken,
		RefreshToken:          tokens.RefreshToken,
		ExpiresAt:             now.Add(time.Duration(tokens.ExpiresIn) * time.Second),
		RefreshTokenExpiresAt: now.Add(time.Duration(tokens.RefreshTokenExpiresIn) * time.Second),
		Scope:                 tokens.Scope,
		User:                  tokens.User,
	}
}

func normalizeBaseURL(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid scheduling API base URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("scheduling API base URL must use http or https")
	}
	loopback := parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1" || parsed.Hostname() == "localhost"
	if parsed.Scheme != "https" && !loopback {
		return "", errors.New("non-loopback scheduling API base URLs must use https")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("scheduling API base URL must not include credentials, query, or fragment")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func randomValue(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate OAuth secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func httpClient(configured *http.Client) *http.Client {
	if configured != nil {
		return configured
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func OpenBrowser(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
