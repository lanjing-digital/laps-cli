package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"
)

func TestLoginUsesBrowserAuthorizationCodePKCEFlow(t *testing.T) {
	var authorizationQuery url.Values
	var tokenForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/oauth/authorize":
			authorizationQuery = r.URL.Query()
			callback, err := url.Parse(authorizationQuery.Get("redirect_uri"))
			if err != nil {
				t.Fatalf("parse callback: %v", err)
			}
			values := callback.Query()
			values.Set("code", "one-time-code")
			values.Set("state", authorizationQuery.Get("state"))
			callback.RawQuery = values.Encode()
			http.Redirect(w, r, callback.String(), http.StatusFound)
		case "/api/auth/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			tokenForm = r.Form
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":             "user-access",
				"token_type":               "Bearer",
				"expires_in":               900,
				"refresh_token":            "user-refresh",
				"refresh_token_expires_in": 86400,
				"scope":                    "schedule:read",
				"user":                     map[string]any{"id": "user_1", "username": "planner", "permissions": []string{"schedule:read"}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := &FileStore{Path: filepath.Join(t.TempDir(), "credentials.json")}
	credentials, err := Login(context.Background(), LoginOptions{
		BaseURL: server.URL,
		Store:   store,
		Timeout: 5 * time.Second,
		OpenBrowser: func(target string) error {
			go func() {
				response, requestErr := http.Get(target)
				if requestErr == nil {
					response.Body.Close()
				}
			}()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if credentials.User.ID != "user_1" || credentials.AccessToken != "user-access" {
		t.Fatalf("unexpected credentials: %#v", credentials)
	}
	if authorizationQuery.Get("response_type") != "code" || authorizationQuery.Get("code_challenge_method") != "S256" {
		t.Fatalf("unexpected authorization query: %v", authorizationQuery)
	}
	if tokenForm.Get("grant_type") != "authorization_code" || tokenForm.Get("code") != "one-time-code" || tokenForm.Get("code_verifier") == "" {
		t.Fatalf("unexpected token form: %v", tokenForm)
	}
}

func TestManagerRefreshesExpiringTokenAndPersistsRotation(t *testing.T) {
	refreshCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/oauth/token" {
			http.NotFound(w, r)
			return
		}
		refreshCalls++
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "old-refresh" {
			t.Fatalf("unexpected refresh form: %v", r.Form)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":             "new-access",
			"token_type":               "Bearer",
			"expires_in":               900,
			"refresh_token":            "new-refresh",
			"refresh_token_expires_in": 86400,
			"scope":                    "schedule:read",
			"user":                     map[string]any{"id": "user_1", "username": "planner"},
		})
	}))
	defer server.Close()

	store := &FileStore{Path: filepath.Join(t.TempDir(), "credentials.json")}
	if err := store.Save(Credentials{
		BaseURL:               server.URL,
		AccessToken:           "old-access",
		RefreshToken:          "old-refresh",
		ExpiresAt:             time.Now().Add(10 * time.Second),
		RefreshTokenExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed credentials: %v", err)
	}
	manager := &Manager{BaseURL: server.URL, Store: store, RefreshSkew: time.Minute}
	token, err := manager.AccessToken(context.Background(), false)
	if err != nil {
		t.Fatalf("access token: %v", err)
	}
	if token != "new-access" || refreshCalls != 1 {
		t.Fatalf("unexpected token=%s refreshCalls=%d", token, refreshCalls)
	}
	updated, err := store.Load()
	if err != nil {
		t.Fatalf("load rotated credentials: %v", err)
	}
	if updated.RefreshToken != "new-refresh" {
		t.Fatalf("refresh token was not rotated: %#v", updated)
	}
}

func TestManagerLogoutRevokesBeforeRemovingCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/oauth/revoke" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse revoke form: %v", err)
		}
		if r.Form.Get("token") != "refresh-to-revoke" || r.Form.Get("client_id") != ClientID {
			t.Fatalf("unexpected revoke form: %v", r.Form)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := &FileStore{Path: filepath.Join(t.TempDir(), "credentials.json")}
	if err := store.Save(Credentials{
		BaseURL:               server.URL,
		AccessToken:           "access",
		RefreshToken:          "refresh-to-revoke",
		ExpiresAt:             time.Now().Add(time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("seed credentials: %v", err)
	}
	manager := &Manager{BaseURL: server.URL, Store: store}
	if err := manager.Logout(context.Background()); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("credentials still exist after logout: %v", err)
	}
}
