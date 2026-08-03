package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreUsesPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "credentials.json")
	store := &FileStore{Path: path}
	want := Credentials{
		BaseURL:               "https://scheduling.example.com",
		AccessToken:           "access",
		RefreshToken:          "refresh",
		ExpiresAt:             time.Now().Add(time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("save credentials: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat credentials: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("credentials permissions = %o, want 600", got)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken {
		t.Fatalf("unexpected credentials: %#v", got)
	}
}
