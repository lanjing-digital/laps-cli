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
	"time"
)

func TestPreviewUsesEnvAndPersistFalse(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"success":true,"plan":{"items":[]},"addedIds":[]}`))
	}))
	defer server.Close()

	t.Setenv(envBaseURL, server.URL)
	t.Setenv(envToken, "env-token")

	var stdout bytes.Buffer
	code := Run([]string{
		"auto-schedule", "preview", "--order-id", "order_1",
		"--capacity-mode", "guaranteed_daily_output", "--resource-id", "team_1",
	}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if gotAuth != "Bearer env-token" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if gotBody["persist"] != false {
		t.Fatalf("expected persist false, got %#v", gotBody["persist"])
	}
	if gotBody["planningReferenceDate"] != defaultPlanningReferenceDate(time.Now()) {
		t.Fatalf("expected default planning reference date today, got %#v", gotBody["planningReferenceDate"])
	}
	orderIDs, ok := gotBody["orderIds"].([]any)
	if !ok || len(orderIDs) != 1 || orderIDs[0] != "order_1" {
		t.Fatalf("unexpected orderIds: %#v", gotBody["orderIds"])
	}
	runOverrides, ok := gotBody["runOverrides"].(map[string]any)
	if !ok || runOverrides["capacityCalculationMode"] != "guaranteed_daily_output" {
		t.Fatalf("unexpected runOverrides: %#v", gotBody["runOverrides"])
	}
	resourceIDs, ok := runOverrides["resourceIds"].([]any)
	if !ok || len(resourceIDs) != 1 || resourceIDs[0] != "team_1" {
		t.Fatalf("unexpected resourceIds: %#v", runOverrides["resourceIds"])
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if output["success"] != true || output["command"] != commandPreview || output["persist"] != false {
		t.Fatalf("unexpected output: %#v", output)
	}
	if output["planningReferenceDate"] != defaultPlanningReferenceDate(time.Now()) {
		t.Fatalf("output must expose the resolved reference date: %#v", output)
	}
}

func TestApplyFlagOverridesEnvAndPersistTrue(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"success":true,"plan":{"items":[]},"addedIds":["sched_1"]}`))
	}))
	defer server.Close()

	t.Setenv(envBaseURL, "http://invalid.local")
	t.Setenv(envToken, "env-token")

	var stdout bytes.Buffer
	code := Run([]string{
		"auto-schedule", "apply",
		"--base-url", server.URL,
		"--token", "flag-token",
		"--plan-id", "plan_1",
		"--order-id", "order_1",
		"--order-id", "order_2",
	}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if gotAuth != "Bearer flag-token" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if gotBody["persist"] != true {
		t.Fatalf("expected persist true, got %#v", gotBody["persist"])
	}
	if gotBody["capacityPlanId"] != "plan_1" {
		t.Fatalf("expected explicit capacity plan, got %#v", gotBody["capacityPlanId"])
	}
	if _, exists := gotBody["planningReferenceDate"]; exists {
		t.Fatalf("explicit plan must not add the default reference date: %#v", gotBody)
	}
	orderIDs := gotBody["orderIds"].([]any)
	if len(orderIDs) != 2 {
		t.Fatalf("unexpected orderIds: %#v", gotBody["orderIds"])
	}
}

func TestCapacityResolvedUsesConfigurationEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/laps/capacity/resolved" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"configuration":{"capacityPlanId":"plan_1"}}`))
	}))
	defer server.Close()
	t.Setenv(envBaseURL, server.URL)
	t.Setenv(envToken, "token")
	var stdout bytes.Buffer
	if code := Run([]string{"capacity", "resolved"}, &stdout, &bytes.Buffer{}); code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "plan_1") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestAutoScheduleRejectsUnknownCapacityMode(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"auto-schedule", "preview", "--capacity-mode", "unknown"}, &stdout, &bytes.Buffer{})
	if code != ExitUsage || !strings.Contains(stdout.String(), "invalid --capacity-mode") {
		t.Fatalf("unexpected result: code=%d output=%s", code, stdout.String())
	}
}

func TestAutoScheduleRejectsInvalidReferenceDate(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"auto-schedule", "preview", "--ref-date", "2026/08/03"}, &stdout, &bytes.Buffer{})
	if code != ExitUsage || !strings.Contains(stdout.String(), "invalid --ref-date") {
		t.Fatalf("unexpected result: code=%d output=%s", code, stdout.String())
	}
}

func TestPreviewSendsSolverOptionsAndExplicitFalse(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"success":true,"plan":{"items":[]}}`))
	}))
	defer server.Close()
	var stdout bytes.Buffer
	code := Run([]string{"auto-schedule", "preview", "--base-url", server.URL, "--token", "token", "--solver-mode", "portfolio", "--include-candidate-plans=false"}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if gotBody["solverMode"] != "portfolio" || gotBody["includeCandidatePlans"] != false {
		t.Fatalf("unexpected solver request: %#v", gotBody)
	}
}

func TestApplyPreviewUsesDedicatedEndpoint(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"success":true,"plan":{"items":[]},"addedIds":["sched_1"]}`))
	}))
	defer server.Close()
	var stdout bytes.Buffer
	code := Run([]string{"auto-schedule", "apply", "--base-url", server.URL, "--token", "token", "--preview-token", "preview_token", "--candidate-solver", "cp-sat"}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if gotPath != "/api/laps/auto-schedule/apply-preview" || gotBody["previewToken"] != "preview_token" || gotBody["candidateSolver"] != "cp-sat" {
		t.Fatalf("unexpected safe apply request: path=%s body=%#v", gotPath, gotBody)
	}
}

func TestApplyPreviewRejectsRecomputeArguments(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"auto-schedule", "apply", "--preview-token", "preview_token", "--candidate-solver", "ga", "--order-id", "order_1"}, &stdout, &bytes.Buffer{})
	if code != ExitUsage || !strings.Contains(stdout.String(), "--preview-token cannot be combined") {
		t.Fatalf("unexpected result: code=%d output=%s", code, stdout.String())
	}
}

func TestMissingOAuthLoginReturnsAuthJSON(t *testing.T) {
	t.Setenv(envToken, "")
	t.Setenv("LAPS_CLI_CONFIG_DIR", t.TempDir())
	var stdout bytes.Buffer
	code := Run([]string{"auto-schedule", "preview"}, &stdout, &bytes.Buffer{})
	if code != ExitAPI {
		t.Fatalf("unexpected exit code: %d", code)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	errObj := output["error"].(map[string]any)
	if errObj["code"] != "AUTH_ERROR" || !strings.Contains(errObj["message"].(string), "laps-cli auth login") {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestHTTPErrorExitCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"bad token"}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	code := Run([]string{"auto-schedule", "preview", "--base-url", server.URL, "--token", "bad"}, &stdout, &bytes.Buffer{})
	if code != ExitHTTP {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	errObj := output["error"].(map[string]any)
	if errObj["code"] != "HTTP_ERROR" || errObj["status"] != float64(http.StatusUnauthorized) {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestActionHelpReturnsTextAndZero(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"auto-schedule", "preview", "--help"}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "laps-cli auto-schedule preview") {
		t.Fatalf("unexpected help output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--plan-id") || !strings.Contains(stdout.String(), "--ref-date") || !strings.Contains(stdout.String(), "defaults to today") {
		t.Fatalf("help must expose capacity plan and default-today reference date: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), `"success"`) {
		t.Fatalf("help should be text, got JSON: %s", stdout.String())
	}
}

func TestTeamsListUsesLapsEndpoint(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("missing bearer header: %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"success":true,"records":[{"id":"team_1"}],"hasMore":false}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	code := Run([]string{"teams", "list", "--base-url", server.URL, "--token", "token"}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if gotPath != "/api/laps/teams" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if output["command"] != "teams list" || output["success"] != true {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestOrdersListSendsQuery(t *testing.T) {
	var gotPath string
	var gotStatus string
	var gotQuery string
	var gotLimit string
	var gotPageToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		values := r.URL.Query()
		gotStatus = values.Get("status")
		gotQuery = values.Get("query")
		gotLimit = values.Get("pageSize")
		gotPageToken = values.Get("pageToken")
		_, _ = w.Write([]byte(`{"success":true,"records":[],"hasMore":false}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	code := Run([]string{
		"orders", "list",
		"--base-url", server.URL,
		"--token", "token",
		"--status", "all",
		"--query", "A001",
		"--limit", "25",
		"--page-token", "50",
	}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if gotPath != "/api/laps/orders" || gotStatus != "all" || gotQuery != "A001" || gotLimit != "25" || gotPageToken != "50" {
		t.Fatalf("unexpected request path/query: path=%s status=%s query=%s limit=%s pageToken=%s", gotPath, gotStatus, gotQuery, gotLimit, gotPageToken)
	}
}

func TestSchedulesListSendsFilters(t *testing.T) {
	var gotPath string
	var gotTeamID string
	var gotOrderID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotTeamID = r.URL.Query().Get("teamId")
		gotOrderID = r.URL.Query().Get("orderId")
		_, _ = w.Write([]byte(`{"success":true,"records":[],"hasMore":false}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	code := Run([]string{
		"schedules", "list",
		"--base-url", server.URL,
		"--token", "token",
		"--team-id", "team_1",
		"--order-id", "order_1",
	}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if gotPath != "/api/laps/schedules" || gotTeamID != "team_1" || gotOrderID != "order_1" {
		t.Fatalf("unexpected request path/query: path=%s team=%s order=%s", gotPath, gotTeamID, gotOrderID)
	}
}

func TestMoveOrderPreviewSendsDryRun(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/laps/move" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"success":true,"dryRun":true,"plan":{"action":"create"}}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	code := Run([]string{
		"move", "order", "preview",
		"--base-url", server.URL,
		"--token", "token",
		"--order-id", "order_1",
		"--to-team-id", "team_1",
	}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if gotBody["sourceType"] != "order" || gotBody["orderId"] != "order_1" || gotBody["targetTeamId"] != "team_1" || gotBody["dryRun"] != true {
		t.Fatalf("unexpected move body: %#v", gotBody)
	}
}

func TestMoveScheduleApplySendsApply(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"success":true,"dryRun":false,"plan":{"action":"update"},"updatedId":"sched_1"}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	code := Run([]string{
		"move", "schedule", "apply",
		"--base-url", server.URL,
		"--token", "token",
		"--schedule-id", "sched_1",
		"--to-team-id", "team_1",
	}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	if gotBody["sourceType"] != "schedule" || gotBody["scheduleId"] != "sched_1" || gotBody["dryRun"] != false {
		t.Fatalf("unexpected move body: %#v", gotBody)
	}
}

func TestEveryLeafHelpReturnsZero(t *testing.T) {
	cases := [][]string{
		{"auth", "login", "--help"},
		{"auth", "status", "--help"},
		{"auth", "logout", "--help"},
		{"teams", "list", "--help"},
		{"orders", "list", "--help"},
		{"orders", "get", "--help"},
		{"orders", "create", "--help"},
		{"orders", "update", "--help"},
		{"orders", "delete", "--help"},
		{"orders", "export", "--help"},
		{"orders", "import", "template", "--help"},
		{"orders", "import", "preview", "--help"},
		{"orders", "import", "apply", "--help"},
		{"schedules", "list", "--help"},
		{"schedules", "get", "--help"},
		{"schedules", "update", "--help"},
		{"schedules", "delete", "--help"},
		{"schedules", "lock", "--help"},
		{"schedules", "apply", "--help"},
		{"materials", "summary", "--help"},
		{"materials", "create", "--help"},
		{"boms", "create", "--help"},
		{"material-import", "template", "--help"},
		{"material-import", "history", "--help"},
		{"material-import", "preview", "--help"},
		{"readiness", "latest", "--help"},
		{"readiness", "analyze", "--help"},
		{"readiness", "external", "import", "--help"},
		{"resources", "apply", "--help"},
		{"efficiencies", "create", "--help"},
		{"calendars", "create", "--help"},
		{"holidays", "create", "--help"},
		{"capacity", "import", "template", "--help"},
		{"capacity", "import", "preview", "--help"},
		{"capacity", "profiles", "--help"},
		{"capacity", "calendar", "days", "--help"},
		{"capacity", "calendar", "range", "set", "--help"},
		{"capacity", "calendar", "category-import", "preview", "--help"},
		{"move", "order", "preview", "--help"},
		{"move", "order", "apply", "--help"},
		{"move", "schedule", "preview", "--help"},
		{"move", "schedule", "apply", "--help"},
	}
	for _, args := range cases {
		var stdout bytes.Buffer
		code := Run(args, &stdout, &bytes.Buffer{})
		if code != ExitOK {
			t.Fatalf("%v unexpected exit code: %d output=%s", args, code, stdout.String())
		}
		if !strings.Contains(stdout.String(), "Usage:") {
			t.Fatalf("%v expected help text, got: %s", args, stdout.String())
		}
		if strings.Contains(stdout.String(), `"success"`) {
			t.Fatalf("%v help should be text, got JSON: %s", args, stdout.String())
		}
	}
}

func TestAutoScheduleCategoryModeAndReadinessControls(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"success":true,"plan":{"items":[]}}`))
	}))
	defer server.Close()
	var stdout bytes.Buffer
	code := Run([]string{
		"auto-schedule", "preview", "--base-url", server.URL, "--token", "token",
		"--capacity-mode", "category_daily_output", "--readiness-enabled", "true",
		"--readiness-mode", "warn", "--readiness-source", "external", "--readiness-max-age-minutes", "60",
		"--prefer-same-product-resource", "false", "--replan-unstarted-orders", "true",
	}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit: %d %s", code, stdout.String())
	}
	overrides := gotBody["runOverrides"].(map[string]any)
	readiness := gotBody["materialReadiness"].(map[string]any)
	if overrides["capacityCalculationMode"] != "category_daily_output" || overrides["preferSameProductResource"] != false || overrides["replanUnstartedOrders"] != true || readiness["mode"] != "warn" || readiness["source"] != "external" || readiness["enabled"] != true || readiness["maxAgeMinutes"] != float64(60) {
		t.Fatalf("unexpected body: %#v", gotBody)
	}
}

func TestCapacityCalendarCommandsUseLapsEndpoints(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}
		_, _ = w.Write([]byte(`{"success":true,"days":[]}`))
	}))
	defer server.Close()
	var stdout bytes.Buffer
	if code := Run([]string{"capacity", "calendar", "range", "set", "--base-url", server.URL, "--token", "token", "--resource-id", "team_1", "--start-date", "2026-08-01", "--end-date", "2026-08-02", "--daily-output", "0", "--reason", "停工"}, &stdout, &bytes.Buffer{}); code != ExitOK {
		t.Fatalf("unexpected exit: %d %s", code, stdout.String())
	}
	if gotMethod != http.MethodPost || gotPath != "/api/laps/capacity-calendar/range" || gotBody["dailyOutput"] != float64(0) || gotBody["reason"] != "停工" {
		t.Fatalf("unexpected range request: %s %s %#v", gotMethod, gotPath, gotBody)
	}
	stdout.Reset()
	if code := Run([]string{"capacity", "calendar", "history", "--base-url", server.URL, "--token", "token", "--limit", "12"}, &stdout, &bytes.Buffer{}); code != ExitOK {
		t.Fatalf("unexpected history exit: %d %s", code, stdout.String())
	}
	if gotMethod != http.MethodGet || gotPath != "/api/laps/capacity-calendar/history" {
		t.Fatalf("unexpected history request: %s %s", gotMethod, gotPath)
	}
}

func TestCapacityCalendarExcelImportUsesMultipartLapsEndpoint(t *testing.T) {
	var gotPath, filename string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		_, header, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		filename = header.Filename
		_, _ = w.Write([]byte(`{"success":true,"preview":{"valid":true}}`))
	}))
	defer server.Close()
	file := filepath.Join(t.TempDir(), "category-capacity.xlsx")
	if err := os.WriteFile(file, []byte("workbook"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if code := Run([]string{"capacity", "calendar", "category-import", "preview", "--base-url", server.URL, "--token", "token", "--file", file}, &stdout, &bytes.Buffer{}); code != ExitOK {
		t.Fatalf("unexpected exit: %d %s", code, stdout.String())
	}
	if gotPath != "/api/laps/capacity-calendar/category-imports/file/preview" || filename != "category-capacity.xlsx" {
		t.Fatalf("unexpected upload: path=%s file=%s", gotPath, filename)
	}
}

func TestResourcesBatchSettingsPreservesFalseAndRejectsMixedInput(t *testing.T) {
	requests := 0
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPatch || r.URL.Path != "/api/laps/resources/factories/batch-settings" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"success":true,"factoryIds":["factory_1"]}`))
	}))
	defer server.Close()
	var stdout bytes.Buffer
	if code := Run([]string{"resources", "batch-settings", "--base-url", server.URL, "--token", "token", "--factory-id", "factory_1", "--enabled", "false", "--is-headquarters", "false"}, &stdout, &bytes.Buffer{}); code != ExitOK {
		t.Fatalf("unexpected exit: %d %s", code, stdout.String())
	}
	if gotBody["enabled"] != false || gotBody["isHeadquarters"] != false {
		t.Fatalf("false values lost: %#v", gotBody)
	}
	file := filepath.Join(t.TempDir(), "batch.json")
	if err := os.WriteFile(file, []byte(`{"factoryIds":["factory_1"],"enabled":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if code := Run([]string{"resources", "batch-settings", "--base-url", server.URL, "--token", "token", "--factory-id", "factory_1", "--file", file}, &stdout, &bytes.Buffer{}); code != ExitUsage || requests != 1 {
		t.Fatalf("mixed input should not request: code=%d requests=%d output=%s", code, requests, stdout.String())
	}
}

func TestScheduleLockAndApplyUseLapsEndpoints(t *testing.T) {
	var paths []string
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"success":true,"updatedIds":["sched_1"]}`))
	}))
	defer server.Close()
	var stdout bytes.Buffer
	if code := Run([]string{"schedules", "lock", "--base-url", server.URL, "--token", "token", "--id", "sched_1", "--locked", "false"}, &stdout, &bytes.Buffer{}); code != ExitOK {
		t.Fatalf("unexpected lock exit: %d %s", code, stdout.String())
	}
	file := filepath.Join(t.TempDir(), "changes.json")
	if err := os.WriteFile(file, []byte(`{"adds":[],"updates":[],"deletes":[],"replan":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if code := Run([]string{"schedules", "apply", "--base-url", server.URL, "--token", "token", "--file", file}, &stdout, &bytes.Buffer{}); code != ExitOK {
		t.Fatalf("unexpected apply exit: %d %s", code, stdout.String())
	}
	if len(paths) != 2 || paths[0] != "PUT /api/laps/schedules/sched_1/lock" || paths[1] != "POST /api/laps/schedules/apply-changes" || bodies[0]["locked"] != false || bodies[1]["replan"] != false {
		t.Fatalf("unexpected requests: paths=%v bodies=%#v", paths, bodies)
	}
}

func TestDeleteRequiresExactIDAndYesBeforeRequest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodDelete || r.URL.Path != "/api/laps/orders/order_1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"deletedIds":["order_1"]}`))
	}))
	defer server.Close()
	var stdout bytes.Buffer
	common := []string{"--base-url", server.URL, "--token", "token", "--id", "order_1"}
	if code := Run(append([]string{"orders", "delete"}, common...), &stdout, &bytes.Buffer{}); code != ExitUsage || requests != 0 {
		t.Fatalf("delete without yes should not request: code=%d requests=%d output=%s", code, requests, stdout.String())
	}
	stdout.Reset()
	if code := Run(append(append([]string{"orders", "delete"}, common...), "--yes"), &stdout, &bytes.Buffer{}); code != ExitOK || requests != 1 {
		t.Fatalf("confirmed delete failed: code=%d requests=%d output=%s", code, requests, stdout.String())
	}
}

func TestSimpleKebabFlagsAndPayloadFileAreMutuallyExclusive(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"success":true,"record":{"id":"material_1"}}`))
	}))
	defer server.Close()
	var stdout bytes.Buffer
	code := Run([]string{"materials", "create", "--base-url", server.URL, "--token", "token", "--code", "MAT-1", "--default-warehouse-code", "WH-1", "--disabled", "false"}, &stdout, &bytes.Buffer{})
	if code != ExitOK || gotBody["code"] != "MAT-1" || gotBody["defaultWarehouseCode"] != "WH-1" || gotBody["disabled"] != false {
		t.Fatalf("unexpected kebab payload: code=%d body=%#v output=%s", code, gotBody, stdout.String())
	}
	input := filepath.Join(t.TempDir(), "material.json")
	if err := os.WriteFile(input, []byte(`{"code":"MAT-2"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	code = Run([]string{"materials", "create", "--base-url", server.URL, "--token", "token", "--file", input, "--code", "MAT-3"}, &stdout, &bytes.Buffer{})
	if code != ExitUsage || !strings.Contains(stdout.String(), "cannot be used together") {
		t.Fatalf("expected file/flags conflict: code=%d output=%s", code, stdout.String())
	}
}

func TestReadPayloadSupportsStdin(t *testing.T) {
	payload, code := readPayload("-", nil, strings.NewReader(`{"styleNo":"A001"}`), &bytes.Buffer{})
	if code != ExitOK || payload["styleNo"] != "A001" {
		t.Fatalf("unexpected stdin payload: code=%d payload=%#v", code, payload)
	}
}

func TestSchedulesTimelineShowsStartAndEndDates(t *testing.T) {
	start := time.Date(2026, 6, 26, 0, 0, 0, 0, time.Local).UnixMilli()
	end := time.Date(2026, 7, 2, 0, 0, 0, 0, time.Local).UnixMilli()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"success": true,
			"records": [{
				"id": "sched_1",
				"orderId": "order_1",
				"teamId": "team_1",
				"teamName": "1组",
				"styleNo": "A001",
				"customerName": "客户A",
				"allocatedQty": 100,
				"efficiency": 0.9,
				"startDate": ` + jsonNumber(start) + `,
				"endDate": ` + jsonNumber(end) + `
			}],
			"hasMore": false
		}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	code := Run([]string{
		"schedules", "list",
		"--base-url", server.URL,
		"--token", "token",
		"--format", "timeline",
	}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	output := stdout.String()
	for _, want := range []string{"Schedule timeline", "1组", "A001/客户A", "start=2026-06-26", "end=2026-07-02"} {
		if !strings.Contains(output, want) {
			t.Fatalf("timeline missing %q: %s", want, output)
		}
	}
}

func TestMovePreviewSVGCanWriteOutputFile(t *testing.T) {
	start := time.Date(2026, 6, 26, 0, 0, 0, 0, time.Local).UnixMilli()
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.Local).UnixMilli()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/laps/schedules" {
			_, _ = w.Write([]byte(`{
				"success": true,
				"records": [{
					"id": "sched_existing",
					"orderId": "order_existing",
					"teamId": "team_1",
					"teamName": "2组",
					"styleNo": "OLD001",
					"customerName": "客户旧",
					"allocatedQty": 60,
					"efficiency": 0.85,
					"startDate": ` + jsonNumber(time.Date(2026, 6, 18, 0, 0, 0, 0, time.Local).UnixMilli()) + `,
					"endDate": ` + jsonNumber(time.Date(2026, 6, 24, 0, 0, 0, 0, time.Local).UnixMilli()) + `
				}],
				"hasMore": false
			}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"success": true,
			"dryRun": true,
			"plan": {
				"sourceType": "order",
				"action": "create",
				"orderId": "order_1",
				"targetTeamId": "team_1",
				"targetTeamName": "2组",
				"allocatedQty": 80,
				"efficiency": 0.9,
				"startDate": ` + jsonNumber(start) + `,
				"endDate": ` + jsonNumber(end) + `
			}
		}`))
	}))
	defer server.Close()

	outputFile := filepath.Join(t.TempDir(), "preview.svg")
	var stdout bytes.Buffer
	code := Run([]string{
		"move", "order", "preview",
		"--base-url", server.URL,
		"--token", "token",
		"--order-id", "order_1",
		"--to-team-id", "team_1",
		"--format", "svg",
		"--output", outputFile,
	}, &stdout, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d output=%s", code, stdout.String())
	}
	raw, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read svg: %v", err)
	}
	content := string(raw)
	for _, want := range []string{"<svg", "2组", "order_1", "2026-06-26", "2026-06-30", "OLD001/客户旧", "2026-06-18", "2026-06-24"} {
		if !strings.Contains(content, want) {
			t.Fatalf("svg missing %q: %s", want, content)
		}
	}
}

func jsonNumber(value int64) string {
	return strings.TrimSpace(string(mustJSON(value)))
}

func mustJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}
