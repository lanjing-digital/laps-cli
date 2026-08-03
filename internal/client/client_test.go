package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type recordingTokenProvider struct {
	tokens []string
	forces []bool
}

func TestDefaultBaseURLUsesProductionHTTPS(t *testing.T) {
	const want = "https://lanjingshuzi.cn:3000"
	if DefaultBaseURL != want {
		t.Fatalf("DefaultBaseURL = %q, want %q", DefaultBaseURL, want)
	}
}

func (p *recordingTokenProvider) AccessToken(_ context.Context, force bool) (string, error) {
	p.forces = append(p.forces, force)
	index := len(p.forces) - 1
	if index >= len(p.tokens) {
		index = len(p.tokens) - 1
	}
	return p.tokens[index], nil
}

func TestAutoScheduleSendsBearerAndBody(t *testing.T) {
	var gotAuth string
	var gotReq AutoScheduleRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/laps/auto-schedule" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"success":true,"plan":{"items":[]}}`))
	}))
	defer server.Close()

	c := New(server.URL, "secret")
	resp, err := c.AutoSchedule(context.Background(), AutoScheduleRequest{
		OrderIDs: []string{"order_1", "order_2"},
		Persist:  true,
		RunOverrides: &AutoScheduleRunOverrides{
			CapacityCalculationMode: "guaranteed_daily_output",
			ResourceIDs:             []string{"team_1"},
		},
	})
	if err != nil {
		t.Fatalf("AutoSchedule returned error: %v", err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if !gotReq.Persist || len(gotReq.OrderIDs) != 2 || gotReq.OrderIDs[0] != "order_1" {
		t.Fatalf("unexpected request body: %#v", gotReq)
	}
	if gotReq.RunOverrides == nil || gotReq.RunOverrides.CapacityCalculationMode != "guaranteed_daily_output" || len(gotReq.RunOverrides.ResourceIDs) != 1 {
		t.Fatalf("unexpected run overrides: %#v", gotReq.RunOverrides)
	}
	if resp["success"] != true {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestGetSendsBearerAndQuery(t *testing.T) {
	var gotAuth string
	var gotQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.Query()
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/laps/orders" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"records":[],"hasMore":false}`))
	}))
	defer server.Close()

	c := New(server.URL, "secret")
	resp, err := c.Get(context.Background(), "/api/laps/orders", url.Values{
		"status":   []string{"pending"},
		"pageSize": []string{"50"},
	})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if gotQuery.Get("status") != "pending" || gotQuery.Get("pageSize") != "50" {
		t.Fatalf("unexpected query: %#v", gotQuery)
	}
	if resp["success"] != true {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestGetSendsBrowserSessionCookieWithoutBearer(t *testing.T) {
	var gotCookie string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"success":true,"records":[]}`))
	}))
	defer server.Close()

	c := NewWithSessionCookie(server.URL, "scheduling_session=session-test")
	if _, err := c.Get(context.Background(), "/api/laps/orders", nil); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if gotCookie != "scheduling_session=session-test" {
		t.Fatalf("unexpected cookie header: %q", gotCookie)
	}
	if gotAuth != "" {
		t.Fatalf("browser session request must not send bearer auth: %q", gotAuth)
	}
}

func TestMoveSendsBearerAndBody(t *testing.T) {
	var gotAuth string
	var gotReq MoveRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/laps/move" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"success":true,"dryRun":true,"plan":{"action":"create"}}`))
	}))
	defer server.Close()

	c := New(server.URL, "secret")
	resp, err := c.Move(context.Background(), MoveRequest{
		SourceType:   "order",
		OrderID:      "order_1",
		TargetTeamID: "team_1",
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("Move returned error: %v", err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if gotReq.SourceType != "order" || gotReq.OrderID != "order_1" || gotReq.TargetTeamID != "team_1" || !gotReq.DryRun {
		t.Fatalf("unexpected request body: %#v", gotReq)
	}
	if resp["success"] != true {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestAutoScheduleHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"bad token"}`))
	}))
	defer server.Close()

	c := New(server.URL, "bad")
	_, err := c.AutoSchedule(context.Background(), AutoScheduleRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr := AsAPIError(err)
	if apiErr.Code != "HTTP_ERROR" || apiErr.Status != http.StatusUnauthorized || apiErr.Message != "bad token" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
}

func TestAutoScheduleParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	c := New(server.URL, "secret")
	_, err := c.AutoSchedule(context.Background(), AutoScheduleRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr := AsAPIError(err)
	if apiErr.Code != "PARSE_ERROR" || apiErr.Status != http.StatusOK {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
}

func TestOAuthProviderRefreshesAndRetriesOnceAfterUnauthorized(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			if got := r.Header.Get("Authorization"); got != "Bearer expired-token" {
				t.Fatalf("unexpected first token: %s", got)
			}
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer refreshed-token" {
			t.Fatalf("unexpected refreshed token: %s", got)
		}
		_, _ = w.Write([]byte(`{"success":true,"records":[]}`))
	}))
	defer server.Close()

	provider := &recordingTokenProvider{tokens: []string{"expired-token", "refreshed-token"}}
	apiClient := NewWithTokenProvider(server.URL, provider)
	if _, err := apiClient.Get(context.Background(), "/api/laps/orders", nil); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if requests != 2 || len(provider.forces) != 2 || provider.forces[0] || !provider.forces[1] {
		t.Fatalf("unexpected refresh sequence: requests=%d forces=%v", requests, provider.forces)
	}
}

func TestJSONVerbsAndRetryReplayRequestBody(t *testing.T) {
	requests := 0
	var methods []string
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		raw, _ := io.ReadAll(r.Body)
		methods = append(methods, r.Method)
		bodies = append(bodies, string(raw))
		if requests == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	provider := &recordingTokenProvider{tokens: []string{"expired", "fresh", "fresh", "fresh"}}
	c := NewWithTokenProvider(server.URL, provider)
	if _, err := c.Patch(context.Background(), "/record/1", map[string]any{"name": "updated"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Put(context.Background(), "/record/1", map[string]any{"name": "replaced"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Delete(context.Background(), "/record/1"); err != nil {
		t.Fatal(err)
	}
	if len(methods) != 4 || methods[0] != http.MethodPatch || methods[1] != http.MethodPatch || methods[2] != http.MethodPut || methods[3] != http.MethodDelete {
		t.Fatalf("unexpected methods: %v", methods)
	}
	if bodies[0] != bodies[1] || !bytes.Contains([]byte(bodies[0]), []byte(`"updated"`)) {
		t.Fatalf("request body was not replayed: %q / %q", bodies[0], bodies[1])
	}
}

func TestUploadAndDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upload":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			raw, _ := io.ReadAll(file)
			if header.Filename != "orders.xlsx" || string(raw) != "workbook" || r.FormValue("mode") != "upsert" {
				t.Fatalf("unexpected multipart input: %s %q %q", header.Filename, raw, r.FormValue("mode"))
			}
			_, _ = w.Write([]byte(`{"success":true}`))
		case "/download":
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			_, _ = w.Write([]byte("template"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "orders.xlsx")
	if err := os.WriteFile(path, []byte("workbook"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := New(server.URL, "token")
	if _, err := c.Upload(context.Background(), "/upload", path, map[string]string{"mode": "upsert"}); err != nil {
		t.Fatal(err)
	}
	raw, contentType, err := c.Download(context.Background(), "/download")
	if err != nil || string(raw) != "template" || contentType == "" {
		t.Fatalf("unexpected download: %q %q %v", raw, contentType, err)
	}
}
