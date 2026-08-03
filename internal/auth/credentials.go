package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const configDirEnv = "LAPS_CLI_CONFIG_DIR"

const (
	credentialsFileName   = "credentials.json"
	fallbackConfigDirName = "laps-cli"
)

var ErrNotLoggedIn = errors.New("not logged in")

type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Permissions []string `json:"permissions"`
}

type Credentials struct {
	BaseURL               string    `json:"baseUrl"`
	AccessToken           string    `json:"accessToken"`
	RefreshToken          string    `json:"refreshToken"`
	ExpiresAt             time.Time `json:"expiresAt"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
	Scope                 string    `json:"scope"`
	User                  User      `json:"user"`
}

type Store interface {
	Load() (Credentials, error)
	Save(Credentials) error
	Remove() error
}

type FileStore struct {
	Path string
}

func DefaultCredentialsPath() (string, error) {
	return resolveDefaultCredentialsPath(os.Getenv, os.UserConfigDir, os.TempDir)
}

// resolveDefaultCredentialsPath keeps credential storage available in minimal
// environments such as containers, where HOME and XDG_CONFIG_HOME may both be
// intentionally unset. LAPS_CLI_CONFIG_DIR remains the preferred explicit
// override, followed by the operating system's normal user config directory.
func resolveDefaultCredentialsPath(
	getenv func(string) string,
	userConfigDir func() (string, error),
	tempDir func() string,
) (string, error) {
	if customDir := strings.TrimSpace(getenv(configDirEnv)); customDir != "" {
		return filepath.Join(customDir, credentialsFileName), nil
	}

	if configDir, err := userConfigDir(); err == nil && strings.TrimSpace(configDir) != "" {
		return filepath.Join(configDir, fallbackConfigDirName, credentialsFileName), nil
	}

	// os.TempDir supplies a platform-native temporary location on macOS, Linux,
	// and Windows. This is a last resort only; users who need durable storage in
	// a headless environment can set LAPS_CLI_CONFIG_DIR explicitly.
	if dir := strings.TrimSpace(tempDir()); dir != "" {
		return filepath.Join(dir, fallbackConfigDirName, credentialsFileName), nil
	}

	return "", fmt.Errorf("resolve credentials directory: user config directory and temporary directory are unavailable")
}

func DefaultStore() (Store, error) {
	path, err := DefaultCredentialsPath()
	if err != nil {
		return nil, err
	}
	return &FileStore{Path: path}, nil
}

func (s *FileStore) Load() (Credentials, error) {
	var credentials Credentials
	raw, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return credentials, ErrNotLoggedIn
	}
	if err != nil {
		return credentials, fmt.Errorf("read credentials: %w", err)
	}
	if err := json.Unmarshal(raw, &credentials); err != nil {
		return credentials, fmt.Errorf("parse credentials: %w", err)
	}
	if credentials.AccessToken == "" || credentials.RefreshToken == "" || credentials.BaseURL == "" {
		return credentials, fmt.Errorf("parse credentials: required fields are missing")
	}
	return credentials, nil
}

func (s *FileStore) Save(credentials Credentials) error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("secure credentials directory: %w", err)
	}
	raw, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credentials: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".credentials-*")
	if err != nil {
		return fmt.Errorf("create credentials file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return fmt.Errorf("secure credentials file: %w", err)
	}
	if _, err := temp.Write(raw); err != nil {
		temp.Close()
		return fmt.Errorf("write credentials: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close credentials file: %w", err)
	}
	if err := os.Rename(tempPath, s.Path); err != nil {
		return fmt.Errorf("replace credentials file: %w", err)
	}
	return nil
}

func (s *FileStore) Remove() error {
	err := os.Remove(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove credentials: %w", err)
	}
	return nil
}
