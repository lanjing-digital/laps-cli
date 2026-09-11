// Package maintenance checks distribution versions without accessing APS data.
package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const CacheTTL = 24 * time.Hour

type Notice struct {
	Current string `json:"current,omitempty"`
	Latest  string `json:"latest,omitempty"`
	Target  string `json:"target,omitempty"`
	Message string `json:"message"`
	Command string `json:"command"`
}

type SkillVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type SkillsState struct {
	Status        string         `json:"status"`
	Directory     string         `json:"directory"`
	Versions      []SkillVersion `json:"versions"`
	BundleVersion string         `json:"bundle_version,omitempty"`
}

type Result struct {
	Success         bool              `json:"success"`
	Current         string            `json:"current_version"`
	Latest          string            `json:"latest_version"`
	UpdateAvailable bool              `json:"update_available"`
	Source          string            `json:"source"`
	Skills          SkillsState       `json:"skills"`
	Notices         map[string]Notice `json:"_notice,omitempty"`
}

type cache struct {
	Latest    string `json:"latest_version"`
	CheckedAt int64  `json:"checked_at"`
	Source    string `json:"source"`
}

type Checker struct {
	Version       string
	ConfigDir     string
	SkillsDir     string
	SkillNames    []string
	Client        *http.Client
	NPMURL        string
	GitHubURL     string
	BundleVersion string
}

func (c Checker) readCache() cache {
	var state cache
	raw, _ := os.ReadFile(filepath.Join(c.ConfigDir, "update-state.json"))
	_ = json.Unmarshal(raw, &state)
	return state
}

func (c Checker) Skills() SkillsState {
	state := SkillsState{Status: "not_installed", Directory: c.SkillsDir, Versions: []SkillVersion{}}
	if c.BundleVersion != "" {
		state.BundleVersion = c.BundleVersion
		state.Status = "in_sync"
		if normalize(c.BundleVersion) != normalize(c.Version) {
			state.Status = "out_of_sync"
		}
		return state
	}
	for _, name := range c.SkillNames {
		if _, err := os.Stat(filepath.Join(c.SkillsDir, name, "SKILL.md")); os.IsNotExist(err) {
			continue
		}
		var marker struct {
			Version string `json:"version"`
		}
		raw, err := os.ReadFile(filepath.Join(c.SkillsDir, name, ".laps-version.json"))
		if err == nil {
			_ = json.Unmarshal(raw, &marker)
		}
		if !validVersion(marker.Version) {
			marker.Version = "unknown"
		}
		state.Versions = append(state.Versions, SkillVersion{Name: name, Version: marker.Version})
		if state.Status == "not_installed" {
			state.Status = "in_sync"
		}
		if normalize(marker.Version) != normalize(c.Version) {
			state.Status = "out_of_sync"
		}
	}
	return state
}

func (c Checker) notices(latest string, skills SkillsState) map[string]Notice {
	notices := map[string]Notice{}
	if IsNewer(latest, c.Version) {
		notices["update"] = Notice{Current: c.Version, Latest: latest, Command: "laps-cli update", Message: fmt.Sprintf("laps-cli %s available, current %s, run: laps-cli update", latest, c.Version)}
	}
	if skills.Status == "out_of_sync" {
		notices["skills"] = Notice{Target: c.Version, Command: "laps-cli update", Message: "laps-cli Skills 版本与 CLI 不一致或无法确认，run: laps-cli update"}
		if skills.BundleVersion != "" {
			notices["skills"] = Notice{Current: skills.BundleVersion, Target: c.Version, Command: "Update the LAPS WorkBuddy connector", Message: "WorkBuddy 连接器 Skills 与 CLI 版本不一致，请更新 WorkBuddy 中的 LAPS 连接器。"}
		}
	}
	return notices
}

func (c Checker) CachedNotices() map[string]Notice {
	return c.notices(c.readCache().Latest, c.Skills())
}

func (c Checker) Check(ctx context.Context, source string) (Result, error) {
	result := Result{Current: c.Version, Skills: c.Skills()}
	if source != "auto" && source != "npm" && source != "github" {
		return result, fmt.Errorf("--source must be auto, npm, or github")
	}
	latest, resolved, err := c.latest(ctx, source)
	if err != nil {
		return result, err
	}
	result.Success, result.Latest, result.Source = true, latest, resolved
	result.UpdateAvailable = IsNewer(latest, c.Version)
	result.Notices = c.notices(latest, result.Skills)
	return result, nil
}

// Refresh is advisory: it cannot change the result of a business command.
func (c Checker) Refresh(ctx context.Context) {
	state := c.readCache()
	if time.Since(time.Unix(state.CheckedAt, 0)) < CacheTTL && validVersion(state.Latest) {
		return
	}
	latest, source, err := c.latest(ctx, "auto")
	if err != nil {
		return
	}
	if err = os.MkdirAll(c.ConfigDir, 0o700); err != nil {
		return
	}
	raw, _ := json.Marshal(cache{Latest: latest, Source: source, CheckedAt: time.Now().Unix()})
	file, err := os.CreateTemp(c.ConfigDir, ".update-state-*")
	if err != nil {
		return
	}
	defer os.Remove(file.Name())
	_, err = file.Write(raw)
	closeErr := file.Close()
	if err == nil && closeErr == nil {
		_ = os.Rename(file.Name(), filepath.Join(c.ConfigDir, "update-state.json"))
	}
}

func (c Checker) latest(ctx context.Context, source string) (string, string, error) {
	if source == "auto" {
		version, _, err := c.latest(ctx, "npm")
		if err == nil {
			return version, "npm", nil
		}
		return c.latest(ctx, "github")
	}
	endpoint := c.NPMURL
	if endpoint == "" {
		endpoint = "https://registry.npmjs.org/@lanjing-digital/laps-cli/latest"
	}
	if source == "github" {
		endpoint = c.GitHubURL
		if endpoint == "" {
			endpoint = "https://api.github.com/repos/lanjing-digital/laps-cli/releases/latest"
		}
	}
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", source, err
	}
	req.Header.Set("User-Agent", "laps-cli-version-check")
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", source, fmt.Errorf("%s version check failed: %w", source, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", source, fmt.Errorf("%s version check: HTTP %d", source, resp.StatusCode)
	}
	var metadata struct {
		Version string `json:"version"`
		Tag     string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256<<10)).Decode(&metadata); err != nil {
		return "", source, fmt.Errorf("%s version check: invalid response", source)
	}
	version := metadata.Version
	if source == "github" {
		version = metadata.Tag
	}
	if !validVersion(version) {
		return "", source, fmt.Errorf("%s version check: invalid version", source)
	}
	return normalize(version), source, nil
}

func normalize(s string) string { return strings.TrimPrefix(strings.TrimSpace(s), "v") }

var semver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

func validVersion(s string) bool { return semver.MatchString(normalize(s)) }

func IsNewer(a, b string) bool {
	aa, bb := semver.FindStringSubmatch(normalize(a)), semver.FindStringSubmatch(normalize(b))
	if aa == nil || bb == nil {
		return false
	}
	for i := 1; i <= 3; i++ {
		x, _ := strconv.ParseUint(aa[i], 10, 64)
		y, _ := strconv.ParseUint(bb[i], 10, 64)
		if x != y {
			return x > y
		}
	}
	if aa[4] == bb[4] {
		return false
	}
	if aa[4] == "" {
		return true
	}
	if bb[4] == "" {
		return false
	}
	xs, ys := strings.Split(aa[4], "."), strings.Split(bb[4], ".")
	for i := 0; i < len(xs) && i < len(ys); i++ {
		if xs[i] == ys[i] {
			continue
		}
		x, xe := strconv.ParseUint(xs[i], 10, 64)
		y, ye := strconv.ParseUint(ys[i], 10, 64)
		if xe == nil && ye == nil {
			return x > y
		}
		if xe == nil || ye == nil {
			return xe != nil
		}
		return xs[i] > ys[i]
	}
	return len(xs) > len(ys)
}
