package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	cliAuth "production-scheduling-cli/internal/auth"
	"production-scheduling-cli/internal/maintenance"
)

func TestStatusReadsPersistedSessionWithoutRefresh(t *testing.T) {
	for _, local := range []bool{false, true} {
		t.Run(map[bool]string{false: "remote", true: "local"}[local], func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != "GET" || r.URL.Path != "/api/laps/me" {
					t.Errorf("status performed a write: %s %s", r.Method, r.URL.Path)
				}
				_, _ = w.Write([]byte(`{"success":true}`))
			}))
			defer server.Close()
			dir := t.TempDir()
			t.Setenv("LAPS_CLI_CONFIG_DIR", dir)
			store := &cliAuth.FileStore{Path: filepath.Join(dir, "credentials.json")}
			credentials := cliAuth.Credentials{BaseURL: server.URL, AccessToken: "test-access", RefreshToken: "test-refresh", ExpiresAt: time.Now().Add(time.Hour), RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour)}
			if local {
				credentials.ExpiresAt = time.Now().Add(-time.Minute)
			}
			if err := store.Save(credentials); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(store.Path)
			info, _ := os.Stat(store.Path)
			args := []string{"auth", "status", "--base-url", server.URL}
			if local {
				args = append(args, "--local")
			}
			for i := 0; i < 3; i++ {
				var out, stderr bytes.Buffer
				if code := RunWithNotices(args, &out, &stderr); code != ExitOK {
					t.Fatalf("status code %d: %s", code, out.String())
				}
				if local && !bytes.Contains(out.Bytes(), []byte(`"authenticated": true`)) {
					t.Fatal(out.String())
				}
			}
			after, _ := os.ReadFile(store.Path)
			afterInfo, _ := os.Stat(store.Path)
			if !bytes.Equal(before, after) || !info.ModTime().Equal(afterInfo.ModTime()) {
				t.Fatal("status modified credentials")
			}
			if local && requests != 0 {
				t.Fatal("local status accessed server")
			}
			if !local && requests != 3 {
				t.Fatal("remote status did not validate account")
			}
		})
	}
}

func TestMissingStatusAndLogoutAreNonInteractive(t *testing.T) {
	t.Setenv("LAPS_CLI_CONFIG_DIR", t.TempDir())
	var out bytes.Buffer
	if code := Run([]string{"auth", "status", "--local"}, &out, &bytes.Buffer{}); code == ExitOK {
		t.Fatal("missing session reported as authenticated")
	}
	out.Reset()
	if code := Run([]string{"auth", "logout"}, &out, &bytes.Buffer{}); code != ExitOK {
		t.Fatalf("logout without session: %d %s", code, out.String())
	}
}

func TestJSONNoticesPreserveResponseAndArtifactBytes(t *testing.T) {
	t.Setenv("LAPS_CLI_NO_SKILLS_NOTIFIER", "")
	t.Setenv("LAPS_CLI_NO_UPDATE_NOTIFIER", "")
	var out bytes.Buffer
	writer := noticeWriter{Writer: &out, checker: maintenance.Checker{Version: "1.0.0", BundleVersion: "0.9.0", ConfigDir: t.TempDir()}}
	writeSuccess(writer, map[string]any{"success": true, "records": []string{"keep"}}, nil)
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["_notice"] == nil || payload["success"] != true || len(payload["records"].([]any)) != 1 {
		t.Fatal(payload)
	}
	out.Reset()
	writeRenderedOutput(writer, "<html>unchanged</html>", "")
	if out.String() != "<html>unchanged</html>" {
		t.Fatal("artifact was decorated")
	}
}
