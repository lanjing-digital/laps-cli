package maintenance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVersionComparison(t *testing.T) {
	for _, tc := range []struct {
		a, b  string
		newer bool
	}{
		{"v1.10.0", "1.9.0", true}, {"1.0.0", "1.0.0-beta.2", true},
		{"1.0.0-beta.10", "1.0.0-beta.2", true}, {"1.0.0-rc.1", "1.0.0", false},
		{"1.0.0+build", "v1.0.0", false}, {"nonsense", "1.0.0", false},
		{"1.0.0", "dev", false}, {"1.0.0", "2.0.0", false},
	} {
		if got := IsNewer(tc.a, tc.b); got != tc.newer {
			t.Errorf("IsNewer(%q,%q)=%v", tc.a, tc.b, got)
		}
	}
}

func TestExplicitCheckDoesNotWriteAndRefreshUsesDailyCache(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "" {
			t.Error("version lookup must not send APS credentials")
		}
		_, _ = w.Write([]byte(`{"version":"1.10.0"}`))
	}))
	defer server.Close()
	checker := Checker{Version: "1.9.0", ConfigDir: t.TempDir(), NPMURL: server.URL}
	result, err := checker.Check(context.Background(), "npm")
	if err != nil || !result.UpdateAvailable || result.Notices["update"].Command != "laps-cli update" {
		t.Fatalf("check: %+v %v", result, err)
	}
	entries, _ := os.ReadDir(checker.ConfigDir)
	if len(entries) != 0 {
		t.Fatal("explicit --check must not write files")
	}
	checker.Refresh(context.Background())
	checker.Refresh(context.Background())
	if calls != 2 {
		t.Fatalf("expected check + one refresh, got %d calls", calls)
	}
	if len(checker.CachedNotices()) != 1 {
		t.Fatal("cached new release was not reported")
	}
	state := cache{Latest: "1.10.0", CheckedAt: time.Now().Add(-25 * time.Hour).Unix()}
	raw, _ := json.Marshal(state)
	_ = os.WriteFile(filepath.Join(checker.ConfigDir, "update-state.json"), raw, 0o600)
	checker.Refresh(context.Background())
	if calls != 3 {
		t.Fatal("expired cache was not refreshed")
	}
}

func TestSourcesAndInvalidMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/github" {
			_, _ = w.Write([]byte(`{"tag_name":"v2.0.0"}`))
			return
		}
		_, _ = w.Write([]byte(`{"version":"invalid"}`))
	}))
	defer server.Close()
	checker := Checker{Version: "1.0.0", ConfigDir: t.TempDir(), NPMURL: server.URL, GitHubURL: server.URL + "/github"}
	if _, err := checker.Check(context.Background(), "npm"); err == nil {
		t.Fatal("invalid registry metadata accepted")
	}
	result, err := checker.Check(context.Background(), "auto")
	if err != nil || result.Source != "github" || result.Latest != "2.0.0" {
		t.Fatalf("auto: %+v %v", result, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := checker.Check(ctx, "github"); err == nil {
		t.Fatal("canceled network lookup succeeded")
	}
}

func TestSkillVersionsAndWorkBuddyBundle(t *testing.T) {
	checker := Checker{Version: "1.2.0", ConfigDir: t.TempDir(), SkillsDir: t.TempDir(), SkillNames: []string{"laps-orders", "laps-capacity"}}
	if checker.Skills().Status != "not_installed" {
		t.Fatal("cold install should not report drift")
	}
	dir := filepath.Join(checker.SkillsDir, "laps-orders")
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("skill"), 0o600)
	if checker.Skills().Status != "out_of_sync" {
		t.Fatal("legacy unversioned skills should be identified")
	}
	_ = os.WriteFile(filepath.Join(dir, ".laps-version.json"), []byte(`{"version":"v1.2.0"}`), 0o600)
	if checker.Skills().Status != "in_sync" {
		t.Fatal("matching skill version should not warn")
	}
	checker.BundleVersion = "1.1.0"
	notice := checker.CachedNotices()["skills"]
	if notice.Current != "1.1.0" || notice.Command != "Update the LAPS WorkBuddy connector" {
		t.Fatalf("bundled skills cannot be repaired by updating CLI alone: %+v", notice)
	}
}
