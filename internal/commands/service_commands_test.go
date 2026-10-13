package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceCommandsRouteEveryLeafAndPreserveConfirmation(t *testing.T) {
	for command, spec := range serviceCommands {
		t.Run(command, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				method := spec.method
				if method == "" {
					method = http.MethodGet
				}
				if r.Method != method || r.URL.Path != strings.ReplaceAll(spec.path, "{id}", "record_1") {
					t.Errorf("wrong request %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-user-token" {
					t.Error("user authentication missing")
				}
				if command == "personal sync apply" {
					var b map[string]any
					_ = json.NewDecoder(r.Body).Decode(&b)
					if b["confirm"] != true || b["previewToken"] != "proof" || b["data"] == nil {
						t.Errorf("invalid sync request %#v", b)
					}
				}
				if command == "personal learning set" {
					var b map[string]any
					_ = json.NewDecoder(r.Body).Decode(&b)
					if b["learningEnabled"] != false {
						t.Error("explicit false lost")
					}
				}
				_, _ = w.Write([]byte(`{"success":true,"persisted":false}`))
			}))
			defer server.Close()
			t.Setenv(envBaseURL, server.URL)
			t.Setenv(envToken, "test-user-token")
			args := strings.Fields(command)
			if spec.id {
				args = append(args, "--id", "record_1")
			}
			if spec.body {
				file := filepath.Join(t.TempDir(), "input.json")
				if err := os.WriteFile(file, []byte(`{"version":1}`), 0600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--file", file)
			}
			if spec.destructive {
				args = append(args, "--yes")
			}
			if command == "personal memories delete" {
				args = append(args, "--version", "1")
			}
			if command == "personal sync apply" {
				args = append(args, "--preview-token", "proof")
			}
			if command == "personal learning set" {
				args = append(args, "--enabled=false")
			}
			var out bytes.Buffer
			if spec.confirm {
				if code := Run(args, &out, &bytes.Buffer{}); code != ExitUsage || calls != 0 {
					t.Fatalf("write escaped confirmation: %d %d %s", code, calls, out.String())
				}
				out.Reset()
				args = append(args, "--confirm")
			}
			if code := Run(args, &out, &bytes.Buffer{}); code != ExitOK || calls != 1 {
				t.Fatalf("route failed %d %d %s", code, calls, out.String())
			}
			var result map[string]any
			_ = json.Unmarshal(out.Bytes(), &result)
			if result["persisted"] != false {
				t.Error("persisted status overwritten")
			}
			out.Reset()
			if code := Run(append(strings.Fields(command), "--help"), &out, &bytes.Buffer{}); code != ExitOK || calls != 1 || !strings.Contains(out.String(), command) {
				t.Fatalf("leaf help failed %d %s", code, out.String())
			}
			if strings.Contains(out.String(), "test-user-token") {
				t.Fatal("help leaked environment credentials")
			}
		})
	}
}
func TestPersonalInputRejectsMissingProofVersionAndAmbiguity(t *testing.T) {
	t.Setenv(envBaseURL, "http://127.0.0.1:1")
	t.Setenv(envToken, "test-user-token")
	cases := [][]string{
		{"personal", "memories", "delete", "--id", "x", "--yes", "--confirm"},
		{"personal", "memories", "clear", "--confirm"},
		{"personal", "memories", "get", "--id", "../other"},
		{"personal", "memories", "get", "--id", ".."},
		{"personal", "sync", "apply", "--set", "schemaVersion=1", "--confirm"},
		{"personal", "memories", "update", "--id", "x", "--set", "title=test", "--confirm"},
		{"personal", "memories", "create", "--file", "x.json", "--set", "title=test", "--confirm"},
		{"personal", "learning", "set", "--confirm"},
	}
	for _, args := range cases {
		var out bytes.Buffer
		if code := Run(args, &out, &bytes.Buffer{}); code != ExitUsage {
			t.Errorf("expected input rejection %v %d %s", args, code, out.String())
		}
	}
}
