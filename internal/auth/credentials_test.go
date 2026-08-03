package auth

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestResolveDefaultCredentialsPath(t *testing.T) {
	configDirErr := errors.New("user config directory unavailable")
	tests := []struct {
		name           string
		configOverride string
		configDir      string
		configDirErr   error
		tempDir        string
		want           string
	}{
		{
			name:           "explicit configuration directory takes precedence",
			configOverride: "/srv/laps-cli",
			configDir:      "/home/tester/.config",
			tempDir:        "/tmp",
			want:           filepath.Join("/srv/laps-cli", credentialsFileName),
		},
		{
			name:      "uses operating system config directory",
			configDir: "/home/tester/.config",
			tempDir:   "/tmp",
			want:      filepath.Join("/home/tester/.config", fallbackConfigDirName, credentialsFileName),
		},
		{
			name:         "falls back when headless environment has no user config directory",
			configDirErr: configDirErr,
			tempDir:      "/tmp",
			want:         filepath.Join("/tmp", fallbackConfigDirName, credentialsFileName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := resolveDefaultCredentialsPath(
				func(name string) string {
					if name == configDirEnv {
						return tt.configOverride
					}
					return ""
				},
				func() (string, error) { return tt.configDir, tt.configDirErr },
				func() string { return tt.tempDir },
			)
			if err != nil {
				t.Fatalf("resolve path: %v", err)
			}
			if path != tt.want {
				t.Fatalf("path = %q, want %q", path, tt.want)
			}
		})
	}
}

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
	if runtime.GOOS != "windows" {
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("credentials permissions = %o, want 600", got)
		}
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken {
		t.Fatalf("unexpected credentials: %#v", got)
	}
}
