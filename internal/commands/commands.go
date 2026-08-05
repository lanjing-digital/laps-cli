package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	cliAuth "production-scheduling-cli/internal/auth"
	"production-scheduling-cli/internal/client"
	"production-scheduling-cli/internal/render"
)

const (
	ExitOK         = 0
	ExitUsage      = 2
	ExitHTTP       = 3
	ExitAPI        = 4
	ExitParse      = 5
	envBaseURL     = "SCHEDULING_API_BASE_URL"
	envToken       = "SCHEDULING_API_TOKEN"
	envCookie      = "SCHEDULING_API_COOKIE"
	commandPreview = "auto-schedule preview"
	commandApply   = "auto-schedule apply"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value != "" {
		*s = append(*s, value)
	}
	return nil
}

type commonFlags struct {
	baseURL *string
	token   *string
}

type renderFlags struct {
	format *string
	output *string
}

type outputPayload struct {
	Success  bool                 `json:"success"`
	Command  string               `json:"command,omitempty"`
	Persist  *bool                `json:"persist,omitempty"`
	OrderIDs []string             `json:"orderIds,omitempty"`
	Error    *client.ErrorPayload `json:"error,omitempty"`
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		writeUsage(stderr)
		return ExitUsage
	}

	switch args[0] {
	case "auth":
		return runAuth(args[1:], stdout, stderr)
	case "auto-schedule":
		return runAutoSchedule(args[1:], stdout, stderr)
	case "capacity":
		return runCapacity(args[1:], stdout, stderr)
	case "teams":
		return runTeams(args[1:], stdout, stderr)
	case "orders":
		return runOrders(args[1:], stdout, stderr)
	case "schedules":
		return runSchedules(args[1:], stdout, stderr)
	case "materials":
		if len(args) > 1 && args[1] == "summary" {
			if wantsHelp(args[2:]) {
				fmt.Fprintln(stdout, "Usage:\n  laps-cli materials summary [--base-url URL]")
				return ExitOK
			}
			fs := flag.NewFlagSet("materials summary", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			common := addCommonFlags(fs)
			if code := parseNoArgs(fs, args[2:], stdout, "materials summary"); code != ExitOK {
				return code
			}
			return runGet(stdout, common, "materials summary", "/api/laps/materials/summary", nil)
		}
		return runCRUD(crudSpec{command: "materials", path: "/api/laps/materials", queryKey: "q", fields: []string{"code", "name", "specification", "unit", "disabled", "default-warehouse-code"}}, args[1:], stdout)
	case "boms":
		return runCRUD(crudSpec{command: "boms", path: "/api/laps/boms", queryKey: "q"}, args[1:], stdout)
	case "material-import":
		return runMaterialImport(args[1:], stdout)
	case "readiness":
		return runReadiness(args[1:], stdout)
	case "resources":
		return runResources(args[1:], stdout)
	case "calendars":
		return runCalendars(args[1:], stdout)
	case "scheduling-policy":
		return runSchedulingPolicy(args[1:], stdout)
	case "efficiencies", "holidays":
		fields := map[string][]string{
			"efficiencies": {"team-id", "team-text", "product-type", "efficiency"},
			"calendars":    {"calendar-code", "calendar-name", "factory", "production-line"},
			"holidays":     {"calendar-id", "holiday-date", "weekday", "label"},
		}
		return runCRUD(crudSpec{command: args[0], path: "/api/laps/" + args[0], queryKey: "q", fields: fields[args[0]]}, args[1:], stdout)
	case "move":
		return runMove(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		writeUsage(stdout)
		return ExitOK
	default:
		writeError(stdout, outputPayload{
			Success: false,
			Error: &client.ErrorPayload{
				Code:    "CONFIG_ERROR",
				Message: "unknown command: " + args[0],
			},
		})
		return ExitUsage
	}
}

func runCapacity(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		writeCapacityUsage(stderr)
		return ExitUsage
	}
	if args[0] == "import" && wantsHelp(args[1:]) && len(args) == 2 {
		writeCapacityUsage(stdout)
		return ExitOK
	}
	if args[0] == "import" && len(args) > 1 && (args[1] == "template" || args[1] == "preview" || args[1] == "apply") {
		return runCapacityImport(args[1:], stdout)
	}
	if args[0] == "profiles" && len(args) > 1 && (args[1] == "get" || args[1] == "apply") {
		return runCapacityProfiles(args[1:], stdout)
	}
	if args[0] == "calendar" {
		return runCapacityCalendar(args[1:], stdout)
	}
	if args[0] == "get" || args[0] == "create" || args[0] == "update" || args[0] == "delete" || args[0] == "profiles" {
		return runCapacityDomain(args, stdout)
	}
	command := "capacity " + args[0]
	switch args[0] {
	case "resolved":
		if wantsHelp(args[1:]) {
			writeCapacityUsage(stdout)
			return ExitOK
		}
		fs := flag.NewFlagSet(command, flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		common := addCommonFlags(fs)
		if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
			return code
		}
		return runGet(stdout, common, command, "/api/laps/capacity/resolved", nil)
	case "list":
		if wantsHelp(args[1:]) {
			writeCapacityUsage(stdout)
			return ExitOK
		}
		fs := flag.NewFlagSet(command, flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		common := addCommonFlags(fs)
		resource := fs.String("resource", "plans", "categories | plans | profiles | capabilities | pools | concurrency-rules | exclusion-groups")
		if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
			return code
		}
		if !validCapacityResource(*resource) {
			return writeConfigError(stdout, "invalid capacity resource: "+*resource)
		}
		return runGet(stdout, common, command, "/api/laps/capacity/"+*resource, nil)
	case "validate", "publish":
		if wantsHelp(args[1:]) {
			writeCapacityUsage(stdout)
			return ExitOK
		}
		fs := flag.NewFlagSet(command, flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		common := addCommonFlags(fs)
		planID := fs.String("plan-id", "", "capacity plan ID")
		if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
			return code
		}
		if strings.TrimSpace(*planID) == "" {
			return writeConfigError(stdout, "--plan-id is required")
		}
		apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
		if !ok {
			return ExitUsage
		}
		response, err := apiClient.Post(context.Background(), "/api/laps/capacity/plans/"+*planID+"/"+args[0], map[string]any{})
		if err != nil {
			return writeAPIError(stdout, err, command, nil, nil)
		}
		writeSuccess(stdout, response, map[string]any{"command": command, "planId": *planID})
		return ExitOK
	case "import":
		fs := flag.NewFlagSet(command, flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		common := addCommonFlags(fs)
		filePath := fs.String("file", "", "normalized capacity-plan JSON file")
		previewOnly := fs.Bool("preview", true, "validate only; set=false to create a draft plan")
		if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
			return code
		}
		if strings.TrimSpace(*filePath) == "" {
			return writeConfigError(stdout, "--file is required")
		}
		raw, err := os.ReadFile(*filePath)
		if err != nil {
			return writeConfigError(stdout, "read import file: "+err.Error())
		}
		var request map[string]any
		if err := json.Unmarshal(raw, &request); err != nil {
			return writeConfigError(stdout, "parse import JSON: "+err.Error())
		}
		apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
		if !ok {
			return ExitUsage
		}
		action := "preview"
		if !*previewOnly {
			action = "apply"
		}
		response, err := apiClient.Post(context.Background(), "/api/laps/capacity/imports/"+action, request)
		if err != nil {
			return writeAPIError(stdout, err, command, nil, nil)
		}
		writeSuccess(stdout, response, map[string]any{"command": command, "preview": *previewOnly, "file": *filePath})
		return ExitOK
	case "-h", "--help", "help":
		writeCapacityUsage(stdout)
		return ExitOK
	default:
		return writeConfigError(stdout, "unknown capacity action: "+args[0])
	}
}

func validCapacityResource(value string) bool {
	switch value {
	case "categories", "plans", "profiles", "capabilities", "pools", "concurrency-rules", "exclusion-groups":
		return true
	default:
		return false
	}
}

func runAuth(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		writeAuthUsage(stderr)
		return ExitUsage
	}
	switch args[0] {
	case "login":
		return runAuthLogin(args[1:], stdout, stderr)
	case "status":
		return runAuthStatus(args[1:], stdout)
	case "logout":
		return runAuthLogout(args[1:], stdout)
	case "-h", "--help", "help":
		writeAuthUsage(stdout)
		return ExitOK
	default:
		return writeConfigError(stdout, "unknown auth action: "+args[0])
	}
}

func runAuthLogin(args []string, stdout io.Writer, stderr io.Writer) int {
	if wantsHelp(args) {
		writeAuthLoginUsage(stdout)
		return ExitOK
	}
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", envOrDefault(envBaseURL, client.DefaultBaseURL), "scheduling API base URL")
	noBrowser := fs.Bool("no-browser", false, "print the authorization URL without opening a browser")
	if code := parseNoArgs(fs, args, stdout, "auth login"); code != ExitOK {
		return code
	}
	store, err := cliAuth.DefaultStore()
	if err != nil {
		return writeAuthError(stdout, "auth login", err)
	}
	credentials, err := cliAuth.Login(context.Background(), cliAuth.LoginOptions{
		BaseURL:   *baseURL,
		Store:     store,
		NoBrowser: *noBrowser,
		Output:    stderr,
	})
	if err != nil {
		return writeAuthError(stdout, "auth login", err)
	}
	writeAuthSuccess(stdout, "auth login", credentials)
	return ExitOK
}

func runAuthStatus(args []string, stdout io.Writer) int {
	if wantsHelp(args) {
		writeAuthStatusUsage(stdout)
		return ExitOK
	}
	fs := flag.NewFlagSet("auth status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", envOrDefault(envBaseURL, client.DefaultBaseURL), "scheduling API base URL")
	if code := parseNoArgs(fs, args, stdout, "auth status"); code != ExitOK {
		return code
	}
	store, err := cliAuth.DefaultStore()
	if err != nil {
		return writeAuthError(stdout, "auth status", err)
	}
	manager := &cliAuth.Manager{BaseURL: *baseURL, Store: store}
	apiClient := client.NewWithTokenProvider(*baseURL, manager)
	response, err := apiClient.Get(context.Background(), "/api/laps/me", nil)
	if err != nil {
		return writeAPIError(stdout, err, "auth status", nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": "auth status"})
	return ExitOK
}

func runAuthLogout(args []string, stdout io.Writer) int {
	if wantsHelp(args) {
		writeAuthLogoutUsage(stdout)
		return ExitOK
	}
	fs := flag.NewFlagSet("auth logout", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", envOrDefault(envBaseURL, client.DefaultBaseURL), "scheduling API base URL")
	if code := parseNoArgs(fs, args, stdout, "auth logout"); code != ExitOK {
		return code
	}
	store, err := cliAuth.DefaultStore()
	if err != nil {
		return writeAuthError(stdout, "auth logout", err)
	}
	manager := &cliAuth.Manager{BaseURL: *baseURL, Store: store}
	if err := manager.Logout(context.Background()); err != nil {
		return writeAuthError(stdout, "auth logout", err)
	}
	writeJSONMap(stdout, map[string]any{"success": true, "command": "auth logout"})
	return ExitOK
}

func runAutoSchedule(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		writeAutoScheduleUsage(stderr)
		return ExitUsage
	}

	switch args[0] {
	case "preview":
		return runAutoScheduleAction(args[1:], false, commandPreview, stdout)
	case "apply":
		return runAutoScheduleAction(args[1:], true, commandApply, stdout)
	case "-h", "--help", "help":
		writeAutoScheduleUsage(stdout)
		return ExitOK
	default:
		writeError(stdout, outputPayload{
			Success: false,
			Error: &client.ErrorPayload{
				Code:    "CONFIG_ERROR",
				Message: "unknown auto-schedule action: " + args[0],
			},
		})
		return ExitUsage
	}
}

func runAutoScheduleAction(args []string, persist bool, command string, stdout io.Writer) int {
	if wantsHelp(args) {
		writeAutoScheduleActionUsage(stdout, command)
		return ExitOK
	}

	var orderIDs stringList
	var resourceIDs stringList
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	view := addRenderFlags(fs)
	fs.Var(&orderIDs, "order-id", "order ID to schedule; repeat for multiple orders")
	fs.Var(&resourceIDs, "resource-id", "temporarily allow only this resource ID; repeat for multiple resources")
	capacityMode := fs.String("capacity-mode", "inherit", "inherit | sam_efficiency | guaranteed_daily_output | category_daily_output")
	preferSameProductResource := fs.String("prefer-same-product-resource", "inherit", "inherit | true | false")
	replanUnstartedOrders := fs.String("replan-unstarted-orders", "inherit", "inherit | true | false")
	readinessEnabled := fs.String("readiness-enabled", "inherit", "inherit | true | false")
	readinessMode := fs.String("readiness-mode", "inherit", "inherit | ignore | warn | block")
	readinessSource := fs.String("readiness-source", "inherit", "inherit | auto | builtin | external")
	readinessMaxAge := fs.Int("readiness-max-age-minutes", 0, "maximum readiness result age")
	solverMode := fs.String("solver-mode", "inherit", "inherit | heuristic | shadow-portfolio | cp-sat | ga | portfolio | hybrid-ga-cp")
	includeCandidatePlans := fs.Bool("include-candidate-plans", true, "return all available solver candidates for preview")
	previewToken := fs.String("preview-token", "", "opaque token returned by auto-schedule preview")
	candidateSolver := fs.String("candidate-solver", "", "solver selected from a preview")
	planID := fs.String("plan-id", "", "capacity plan ID (required by server for auto-schedule)")
	refDate := fs.String("ref-date", "", "planning reference date YYYY-MM-DD; defaults to today when --plan-id is omitted")
	if code := parseNoArgs(fs, args, stdout, command); code != ExitOK {
		return code
	}
	resolvedRefDate := strings.TrimSpace(*refDate)
	if strings.TrimSpace(*planID) == "" && resolvedRefDate == "" {
		resolvedRefDate = defaultPlanningReferenceDate(time.Now())
	}
	if resolvedRefDate != "" {
		if _, err := time.Parse("2006-01-02", resolvedRefDate); err != nil {
			return writeConfigError(stdout, "invalid --ref-date: expected YYYY-MM-DD")
		}
	}
	if !oneOf(*capacityMode, "inherit", "sam_efficiency", "guaranteed_daily_output", "category_daily_output") {
		return writeConfigError(stdout, "invalid --capacity-mode: "+*capacityMode)
	}
	if !oneOf(*preferSameProductResource, "inherit", "true", "false") {
		return writeConfigError(stdout, "invalid --prefer-same-product-resource: "+*preferSameProductResource)
	}
	if !oneOf(*replanUnstartedOrders, "inherit", "true", "false") {
		return writeConfigError(stdout, "invalid --replan-unstarted-orders: "+*replanUnstartedOrders)
	}
	if !oneOf(*readinessEnabled, "inherit", "true", "false") {
		return writeConfigError(stdout, "invalid --readiness-enabled: "+*readinessEnabled)
	}
	if !oneOf(*readinessMode, "inherit", "ignore", "warn", "block") {
		return writeConfigError(stdout, "invalid --readiness-mode: "+*readinessMode)
	}
	if !oneOf(*readinessSource, "inherit", "auto", "builtin", "external") {
		return writeConfigError(stdout, "invalid --readiness-source: "+*readinessSource)
	}
	if !oneOf(*solverMode, "inherit", "heuristic", "shadow-portfolio", "cp-sat", "ga", "portfolio", "hybrid-ga-cp") {
		return writeConfigError(stdout, "invalid --solver-mode: "+*solverMode)
	}
	if strings.TrimSpace(*previewToken) != "" {
		if !persist {
			return writeConfigError(stdout, "--preview-token can only be used with auto-schedule apply")
		}
		if !oneOf(*candidateSolver, "heuristic", "cp-sat", "ga", "hybrid-ga-cp") {
			return writeConfigError(stdout, "--candidate-solver must be heuristic, cp-sat, ga, or hybrid-ga-cp")
		}
		forbidden := map[string]bool{}
		fs.Visit(func(item *flag.Flag) { forbidden[item.Name] = true })
		for _, name := range []string{"order-id", "resource-id", "plan-id", "ref-date", "capacity-mode", "prefer-same-product-resource", "replan-unstarted-orders", "readiness-enabled", "readiness-mode", "readiness-source", "readiness-max-age-minutes", "solver-mode", "include-candidate-plans"} {
			if forbidden[name] {
				return writeConfigError(stdout, "--preview-token cannot be combined with "+"--"+name)
			}
		}
		apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
		if !ok {
			return ExitUsage
		}
		apiClient.HTTPClient.Timeout = 120 * time.Second
		response, err := apiClient.ApplyAutoSchedulePreview(context.Background(), client.ApplyAutoSchedulePreviewRequest{
			PreviewToken: strings.TrimSpace(*previewToken), CandidateSolver: *candidateSolver,
		})
		if err != nil {
			return writeAPIError(stdout, err, command, nil, nil)
		}
		return emitVisualResponse(stdout, apiClient, response, map[string]any{"command": command, "candidateSolver": *candidateSolver}, view)
	}

	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	request := client.AutoScheduleRequest{
		OrderIDs:              orderIDs,
		Persist:               persist,
		CapacityPlanId:        strings.TrimSpace(*planID),
		PlanningReferenceDate: resolvedRefDate,
	}
	if *solverMode != "inherit" {
		request.SolverMode = *solverMode
	}
	if !persist {
		request.IncludeCandidatePlans = includeCandidatePlans
	}
	if *capacityMode != "inherit" || len(resourceIDs) > 0 || *preferSameProductResource != "inherit" || *replanUnstartedOrders != "inherit" {
		request.RunOverrides = &client.AutoScheduleRunOverrides{
			ResourceIDs: resourceIDs,
		}
		if *capacityMode != "inherit" {
			request.RunOverrides.CapacityCalculationMode = *capacityMode
		}
		if *preferSameProductResource != "inherit" {
			value := *preferSameProductResource == "true"
			request.RunOverrides.PreferSameProductResource = &value
		}
		if *replanUnstartedOrders != "inherit" {
			value := *replanUnstartedOrders == "true"
			request.RunOverrides.ReplanUnstartedOrders = &value
		}
	}
	if *readinessEnabled != "inherit" || *readinessMode != "inherit" || *readinessSource != "inherit" || *readinessMaxAge > 0 {
		control := &client.MaterialReadinessControl{MaxAgeMinutes: *readinessMaxAge}
		if *readinessEnabled != "inherit" {
			value := *readinessEnabled == "true"
			control.Enabled = &value
		}
		if *readinessMode != "inherit" {
			control.Mode = *readinessMode
		}
		if *readinessSource != "inherit" {
			control.Source = *readinessSource
		}
		request.MaterialReadiness = control
	}
	apiClient.HTTPClient.Timeout = 120 * time.Second
	response, err := apiClient.AutoSchedule(context.Background(), request)
	if err != nil {
		return writeAPIError(stdout, err, command, &persist, orderIDs)
	}

	metadata := map[string]any{
		"command":  command,
		"persist":  persist,
		"orderIds": []string(orderIDs),
	}
	if capacityPlanID := strings.TrimSpace(*planID); capacityPlanID != "" {
		metadata["capacityPlanId"] = capacityPlanID
	}
	if resolvedRefDate != "" {
		metadata["planningReferenceDate"] = resolvedRefDate
	}
	return emitVisualResponse(stdout, apiClient, response, metadata, view)
}

func defaultPlanningReferenceDate(now time.Time) string {
	return now.Format("2006-01-02")
}

func runTeams(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		writeTeamsUsage(stderr)
		return ExitUsage
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		writeTeamsUsage(stdout)
		return ExitOK
	}
	if args[0] != "list" {
		return writeConfigError(stdout, "unknown teams action: "+args[0])
	}
	if wantsHelp(args[1:]) {
		writeTeamsListUsage(stdout)
		return ExitOK
	}

	fs := flag.NewFlagSet("teams list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	if code := parseNoArgs(fs, args[1:], stdout, "teams list"); code != ExitOK {
		return code
	}
	return runGet(stdout, common, "teams list", "/api/laps/teams", nil)
}

func runOrders(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		writeOrdersUsage(stderr)
		return ExitUsage
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		writeOrdersUsage(stdout)
		return ExitOK
	}
	if args[0] != "list" {
		return runOrdersExtended(args, stdout)
	}
	if wantsHelp(args[1:]) {
		writeOrdersListUsage(stdout)
		return ExitOK
	}

	fs := flag.NewFlagSet("orders list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	status := fs.String("status", "pending", "pending | scheduled | all")
	query := fs.String("query", "", "filter by order ID, style number, style description, or customer")
	limit := fs.Int("limit", 100, "page size")
	pageToken := fs.String("page-token", "", "pagination token")
	if code := parseNoArgs(fs, args[1:], stdout, "orders list"); code != ExitOK {
		return code
	}

	values := url.Values{}
	values.Set("status", *status)
	if *query != "" {
		values.Set("query", *query)
	}
	values.Set("pageSize", fmt.Sprint(*limit))
	if *pageToken != "" {
		values.Set("pageToken", *pageToken)
	}
	return runGet(stdout, common, "orders list", "/api/laps/orders", values)
}

func runSchedules(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		writeSchedulesUsage(stderr)
		return ExitUsage
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		writeSchedulesUsage(stdout)
		return ExitOK
	}
	switch args[0] {
	case "lock":
		return runScheduleLock(args[1:], stdout)
	case "apply":
		return runScheduleApply(args[1:], stdout)
	}
	if args[0] != "list" {
		return runCRUD(crudSpec{command: "schedules", path: "/api/laps/schedules", queryKey: "query", fields: []string{"allocated-qty", "team-id", "order-id", "efficiency", "start-date", "actual-end-date", "notify-completed", "sewing-days"}}, args, stdout)
	}
	if wantsHelp(args[1:]) {
		writeSchedulesListUsage(stdout)
		return ExitOK
	}

	fs := flag.NewFlagSet("schedules list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	teamID := fs.String("team-id", "", "team ID or team name")
	orderID := fs.String("order-id", "", "order ID or style number")
	limit := fs.Int("limit", 100, "page size")
	pageToken := fs.String("page-token", "", "pagination token")
	view := addRenderFlags(fs)
	if code := parseNoArgs(fs, args[1:], stdout, "schedules list"); code != ExitOK {
		return code
	}

	values := url.Values{}
	if *teamID != "" {
		values.Set("teamId", *teamID)
	}
	if *orderID != "" {
		values.Set("orderId", *orderID)
	}
	values.Set("pageSize", fmt.Sprint(*limit))
	if *pageToken != "" {
		values.Set("pageToken", *pageToken)
	}
	apiClient, ok := newAPIClient(common, stdout, "schedules list", nil, nil)
	if !ok {
		return ExitUsage
	}
	response, err := apiClient.Get(context.Background(), "/api/laps/schedules", values)
	if err != nil {
		return writeAPIError(stdout, err, "schedules list", nil, nil)
	}
	return emitResponse(stdout, response, map[string]any{"command": "schedules list"}, view)
}

func runScheduleLock(args []string, stdout io.Writer) int {
	command := "schedules lock"
	if wantsHelp(args) {
		fmt.Fprintln(stdout, "Usage:\n  laps-cli schedules lock --id ID --locked true|false")
		return ExitOK
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	id := fs.String("id", "", "schedule ID")
	locked := fs.String("locked", "", "true | false")
	if code := parseNoArgs(fs, args, stdout, command); code != ExitOK {
		return code
	}
	if *id == "" || *locked == "" {
		return writeConfigError(stdout, "--id and --locked are required")
	}
	value, err := strconv.ParseBool(*locked)
	if err != nil {
		return writeConfigError(stdout, "--locked must be true or false")
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	response, err := apiClient.Put(context.Background(), "/api/laps/schedules/"+url.PathEscape(*id)+"/lock", map[string]any{"locked": value})
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command, "id": *id})
	return ExitOK
}

func runScheduleApply(args []string, stdout io.Writer) int {
	command := "schedules apply"
	if wantsHelp(args) {
		fmt.Fprintln(stdout, "Usage:\n  laps-cli schedules apply --file JSON")
		return ExitOK
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	filePath := fs.String("file", "", "apply-changes JSON file or -")
	if code := parseNoArgs(fs, args, stdout, command); code != ExitOK {
		return code
	}
	payload, code := readPayload(*filePath, nil, os.Stdin, stdout)
	if code != ExitOK {
		return code
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	response, err := apiClient.Post(context.Background(), "/api/laps/schedules/apply-changes", payload)
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}

func runMove(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		writeMoveUsage(stderr)
		return ExitUsage
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		writeMoveUsage(stdout)
		return ExitOK
	}
	switch args[0] {
	case "order":
		return runMoveAction(args[1:], "order", stdout)
	case "schedule":
		return runMoveAction(args[1:], "schedule", stdout)
	default:
		return writeConfigError(stdout, "unknown move source: "+args[0])
	}
}

func runMoveAction(args []string, sourceType string, stdout io.Writer) int {
	if len(args) == 0 {
		writeMoveSourceUsage(stdout, sourceType)
		return ExitUsage
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		writeMoveSourceUsage(stdout, sourceType)
		return ExitOK
	}

	var dryRun bool
	switch args[0] {
	case "preview":
		dryRun = true
	case "apply":
		dryRun = false
	default:
		return writeConfigError(stdout, "unknown move action: "+args[0])
	}
	command := "move " + sourceType + " " + args[0]
	if wantsHelp(args[1:]) {
		writeMoveActionUsage(stdout, sourceType, args[0])
		return ExitOK
	}

	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	orderID := fs.String("order-id", "", "order ID")
	scheduleID := fs.String("schedule-id", "", "schedule ID")
	targetTeamID := fs.String("to-team-id", "", "target team ID")
	view := addRenderFlags(fs)
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	if *targetTeamID == "" || (sourceType == "order" && *orderID == "") || (sourceType == "schedule" && *scheduleID == "") {
		return writeConfigError(stdout, "missing required move flags")
	}

	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	response, err := apiClient.Move(context.Background(), client.MoveRequest{
		SourceType:   sourceType,
		OrderID:      *orderID,
		ScheduleID:   *scheduleID,
		TargetTeamID: *targetTeamID,
		DryRun:       dryRun,
	})
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	return emitVisualResponse(stdout, apiClient, response, map[string]any{"command": command}, view)
}

func runGet(stdout io.Writer, common commonFlags, command string, path string, values url.Values) int {
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	response, err := apiClient.Get(context.Background(), path, values)
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}

func addCommonFlags(fs *flag.FlagSet) commonFlags {
	return commonFlags{
		baseURL: fs.String("base-url", envOrDefault(envBaseURL, client.DefaultBaseURL), "scheduling API base URL"),
		token:   fs.String("token", os.Getenv(envToken), "scheduling API bearer token"),
	}
}

func addRenderFlags(fs *flag.FlagSet) renderFlags {
	return renderFlags{
		format: fs.String("format", "json", "output format: json | timeline | svg | html"),
		output: fs.String("output", "", "write rendered timeline/svg/html output to a file"),
	}
}

func parseNoArgs(fs *flag.FlagSet, args []string, stdout io.Writer, command string) int {
	if err := fs.Parse(args); err != nil {
		writeError(stdout, outputPayload{
			Success: false,
			Command: command,
			Error: &client.ErrorPayload{
				Code:    "CONFIG_ERROR",
				Message: err.Error(),
			},
		})
		return ExitUsage
	}
	if fs.NArg() > 0 {
		writeError(stdout, outputPayload{
			Success: false,
			Command: command,
			Error: &client.ErrorPayload{
				Code:    "CONFIG_ERROR",
				Message: "unexpected positional arguments: " + strings.Join(fs.Args(), " "),
			},
		})
		return ExitUsage
	}
	return ExitOK
}

func newAPIClient(common commonFlags, stdout io.Writer, command string, persist *bool, orderIDs []string) (*client.Client, bool) {
	if strings.TrimSpace(*common.baseURL) == "" {
		writeError(stdout, outputPayload{
			Success:  false,
			Command:  command,
			Persist:  persist,
			OrderIDs: orderIDs,
			Error: &client.ErrorPayload{
				Code:    "CONFIG_ERROR",
				Message: "--base-url is required",
			},
		})
		return nil, false
	}
	if strings.TrimSpace(*common.token) != "" {
		return client.New(*common.baseURL, *common.token), true
	}
	if sessionCookie := strings.TrimSpace(os.Getenv(envCookie)); sessionCookie != "" {
		return client.NewWithSessionCookie(*common.baseURL, sessionCookie), true
	}
	store, err := cliAuth.DefaultStore()
	if err != nil {
		writeAuthError(stdout, command, err)
		return nil, false
	}
	manager := &cliAuth.Manager{BaseURL: *common.baseURL, Store: store}
	return client.NewWithTokenProvider(*common.baseURL, manager), true
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value != "" {
		return value
	}
	return fallback
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func exitCodeFor(code string) int {
	switch code {
	case "HTTP_ERROR":
		return ExitHTTP
	case "PARSE_ERROR":
		return ExitParse
	case "CONFIG_ERROR":
		return ExitUsage
	default:
		return ExitAPI
	}
}

func writeUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli auth login [--base-url URL]")
	fmt.Fprintln(w, "  laps-cli auth status [--base-url URL]")
	fmt.Fprintln(w, "  laps-cli auth logout")
	fmt.Fprintln(w, "  laps-cli teams list [--base-url URL]")
	fmt.Fprintln(w, "  laps-cli orders list|get|create|update|delete|export")
	fmt.Fprintln(w, "  laps-cli orders import template|preview|apply")
	fmt.Fprintln(w, "  laps-cli schedules list|get|update|delete|lock|apply")
	fmt.Fprintln(w, "  laps-cli auto-schedule preview [--order-id ID ...]")
	fmt.Fprintln(w, "  laps-cli auto-schedule apply --preview-token TOKEN --candidate-solver SOLVER")
	fmt.Fprintln(w, "  laps-cli capacity resolved")
	fmt.Fprintln(w, "  laps-cli capacity list [--resource plans|categories|profiles|capabilities]")
	fmt.Fprintln(w, "  laps-cli capacity validate --plan-id ID")
	fmt.Fprintln(w, "  laps-cli capacity publish --plan-id ID")
	fmt.Fprintln(w, "  laps-cli capacity profiles get|apply --factory-id ID")
	fmt.Fprintln(w, "  laps-cli capacity import template|preview|apply")
	fmt.Fprintln(w, "  laps-cli capacity calendar days|range|category-days|category-range|category-import|history")
	fmt.Fprintln(w, "  laps-cli capacity import --file normalized.json [--preview=false]  # legacy alias")
	fmt.Fprintln(w, "  laps-cli move order preview --order-id ID --to-team-id ID")
	fmt.Fprintln(w, "  laps-cli move order apply --order-id ID --to-team-id ID")
	fmt.Fprintln(w, "  laps-cli move schedule preview --schedule-id ID --to-team-id ID")
	fmt.Fprintln(w, "  laps-cli move schedule apply --schedule-id ID --to-team-id ID")
	fmt.Fprintln(w, "  laps-cli scheduling-policy list|get|create|update|delete|clone|validate|publish|runs")
	writeDomainUsage(w)
}

func writeAuthUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli auth login [--base-url URL] [--no-browser]")
	fmt.Fprintln(w, "  laps-cli auth status [--base-url URL]")
	fmt.Fprintln(w, "  laps-cli auth logout")
}

func writeAuthLoginUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli auth login [--base-url URL] [--no-browser]")
	fmt.Fprintln(w, "  Opens the system browser and reuses the current application login session.")
}

func writeAuthStatusUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli auth status [--base-url URL]")
}

func writeAuthLogoutUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli auth logout")
}

func writeAutoScheduleUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli auto-schedule preview [--base-url URL] [--order-id ID ...] [--plan-id ID | --ref-date YYYY-MM-DD] [--solver-mode MODE]")
	fmt.Fprintln(w, "  laps-cli auto-schedule apply --preview-token TOKEN --candidate-solver SOLVER")
	writeEnvironment(w)
}

func writeCapacityUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli capacity resolved")
	fmt.Fprintln(w, "  laps-cli capacity list --resource plans|categories|profiles|capabilities|pools|concurrency-rules|exclusion-groups")
	fmt.Fprintln(w, "  laps-cli capacity validate --plan-id ID")
	fmt.Fprintln(w, "  laps-cli capacity publish --plan-id ID")
	fmt.Fprintln(w, "  laps-cli capacity profiles get|apply --factory-id ID")
	fmt.Fprintln(w, "  laps-cli capacity import template|preview|apply")
	fmt.Fprintln(w, "  laps-cli capacity import --file normalized.json [--preview=true|false]  # legacy alias")
	writeEnvironment(w)
}

func writeAutoScheduleActionUsage(w io.Writer, command string) {
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  laps-cli %s [--base-url URL] [--order-id ID ...] [--plan-id ID | --ref-date YYYY-MM-DD] [--capacity-mode MODE] [--resource-id ID ...] [--format json|timeline|svg|html] [--output FILE]\n", command)
	writeCommonFlags(w)
	writeRenderFlags(w)
	fmt.Fprintln(w, "  --plan-id ID                Use this published capacity plan")
	fmt.Fprintln(w, "  --ref-date YYYY-MM-DD       Resolve the published plan for this date; defaults to today when --plan-id is omitted")
	fmt.Fprintln(w, "  --capacity-mode VALUE       inherit | sam_efficiency | guaranteed_daily_output | category_daily_output")
	fmt.Fprintln(w, "  --resource-id ID            Repeat to constrain candidates for this run")
	fmt.Fprintln(w, "  --prefer-same-product-resource VALUE  inherit | true | false")
	fmt.Fprintln(w, "  --replan-unstarted-orders VALUE        inherit | true | false")
	fmt.Fprintln(w, "  --readiness-enabled VALUE   inherit | true | false")
	fmt.Fprintln(w, "  --readiness-mode VALUE      inherit | ignore | warn | block")
	fmt.Fprintln(w, "  --readiness-source VALUE    inherit | auto | builtin | external")
	fmt.Fprintln(w, "  --solver-mode VALUE          inherit | heuristic | shadow-portfolio | cp-sat | ga | portfolio | hybrid-ga-cp")
	fmt.Fprintln(w, "  --include-candidate-plans    true | false; preview defaults to true")
	fmt.Fprintln(w, "  --preview-token TOKEN        apply exactly one previously previewed candidate")
	fmt.Fprintln(w, "  --candidate-solver VALUE     required with --preview-token: heuristic | cp-sat | ga | hybrid-ga-cp")
	fmt.Fprintln(w, "Output:")
	fmt.Fprintln(w, "  Default JSON is stable for AI agents. HTML is the preferred readable Gantt view; SVG is a portable fallback.")
}

func writeTeamsUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli teams list [--base-url URL]")
}

func writeTeamsListUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli teams list [--base-url URL]")
	writeCommonFlags(w)
}

func writeOrdersUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli orders list [--base-url URL] [--status pending|scheduled|all] [--query TEXT] [--limit N] [--page-token TOKEN]")
}

func writeOrdersListUsage(w io.Writer) {
	writeOrdersUsage(w)
	writeCommonFlags(w)
	fmt.Fprintln(w, "  --status VALUE     pending | scheduled | all (default pending)")
	fmt.Fprintln(w, "  --query TEXT       filter by order ID, style number, style description, or customer")
	fmt.Fprintln(w, "  --limit N          page size")
	fmt.Fprintln(w, "  --page-token TOKEN pagination token")
}

func writeSchedulesUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli schedules list [--base-url URL] [--team-id ID] [--order-id ID] [--limit N] [--page-token TOKEN] [--format json|timeline|svg|html] [--output FILE]")
}

func writeSchedulesListUsage(w io.Writer) {
	writeSchedulesUsage(w)
	writeCommonFlags(w)
	fmt.Fprintln(w, "  --team-id ID       team ID or team name")
	fmt.Fprintln(w, "  --order-id ID      order ID or style number")
	fmt.Fprintln(w, "  --limit N          page size")
	fmt.Fprintln(w, "  --page-token TOKEN pagination token")
	writeRenderFlags(w)
}

func writeMoveUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli move order preview --order-id ID --to-team-id ID")
	fmt.Fprintln(w, "  laps-cli move order apply --order-id ID --to-team-id ID")
	fmt.Fprintln(w, "  laps-cli move schedule preview --schedule-id ID --to-team-id ID")
	fmt.Fprintln(w, "  laps-cli move schedule apply --schedule-id ID --to-team-id ID")
}

func writeMoveSourceUsage(w io.Writer, sourceType string) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  laps-cli move %s preview [flags]\n", sourceType)
	fmt.Fprintf(w, "  laps-cli move %s apply [flags]\n", sourceType)
}

func writeMoveActionUsage(w io.Writer, sourceType string, action string) {
	fmt.Fprintln(w, "Usage:")
	if sourceType == "order" {
		fmt.Fprintf(w, "  laps-cli move order %s --order-id ID --to-team-id ID [--base-url URL] [--format json|timeline|svg|html] [--output FILE]\n", action)
	} else {
		fmt.Fprintf(w, "  laps-cli move schedule %s --schedule-id ID --to-team-id ID [--base-url URL] [--format json|timeline|svg|html] [--output FILE]\n", action)
	}
	writeCommonFlags(w)
	fmt.Fprintln(w, "  --order-id ID      order ID (move order only)")
	fmt.Fprintln(w, "  --schedule-id ID   schedule ID (move schedule only)")
	fmt.Fprintln(w, "  --to-team-id ID    target team ID")
	writeRenderFlags(w)
}

func writeCommonFlags(w io.Writer) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintf(w, "  --base-url URL     scheduling API base URL; default %s or SCHEDULING_API_BASE_URL\n", client.DefaultBaseURL)
	fmt.Fprintln(w, "  --token TOKEN      explicit OAuth access token override; normally use `laps-cli auth login`")
	fmt.Fprintln(w, "  -h, --help         help for this command")
	fmt.Fprintln(w, "")
}

func writeRenderFlags(w io.Writer) {
	fmt.Fprintln(w, "  --format VALUE     json | timeline | svg | html (default json)")
	fmt.Fprintln(w, "  --output FILE      write timeline/svg/html output to a file")
}

func writeEnvironment(w io.Writer) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Environment:")
	fmt.Fprintf(w, "  SCHEDULING_API_BASE_URL  default %s\n", client.DefaultBaseURL)
	fmt.Fprintln(w, "  SCHEDULING_API_TOKEN     optional OAuth access token override (no automatic refresh)")
}

func writeAuthError(w io.Writer, command string, err error) int {
	writeJSONMap(w, map[string]any{
		"success": false,
		"command": command,
		"error":   map[string]any{"code": "AUTH_ERROR", "message": err.Error()},
	})
	return ExitAPI
}

func writeAuthSuccess(w io.Writer, command string, credentials cliAuth.Credentials) {
	writeJSONMap(w, map[string]any{
		"success":               true,
		"command":               command,
		"expiresAt":             credentials.ExpiresAt,
		"refreshTokenExpiresAt": credentials.RefreshTokenExpiresAt,
		"scope":                 credentials.Scope,
		"user":                  credentials.User,
	})
}

func writeJSONMap(w io.Writer, payload map[string]any) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			return true
		}
	}
	return false
}

func writeConfigError(stdout io.Writer, message string) int {
	writeError(stdout, outputPayload{
		Success: false,
		Error: &client.ErrorPayload{
			Code:    "CONFIG_ERROR",
			Message: message,
		},
	})
	return ExitUsage
}

func writeAPIError(stdout io.Writer, err error, command string, persist *bool, orderIDs []string) int {
	apiErr := client.AsAPIError(err)
	code := exitCodeFor(apiErr.Code)
	writeError(stdout, outputPayload{
		Success:  false,
		Command:  command,
		Persist:  persist,
		OrderIDs: orderIDs,
		Error: &client.ErrorPayload{
			Code:    apiErr.Code,
			Message: apiErr.Message,
			Status:  apiErr.Status,
		},
	})
	return code
}

func writeError(w io.Writer, payload outputPayload) {
	writeJSON(w, payload)
}

func writeSuccess(w io.Writer, response map[string]any, metadata map[string]any) {
	payload := mergePayload(response, metadata)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

func emitResponse(w io.Writer, response map[string]any, metadata map[string]any, view renderFlags) int {
	format := strings.ToLower(strings.TrimSpace(*view.format))
	if format == "" {
		format = "json"
	}
	payload := mergePayload(response, metadata)
	switch format {
	case "json":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(payload)
	case "timeline":
		return writeRenderedOutput(w, render.Timeline(payload), *view.output)
	case "svg":
		return writeRenderedOutput(w, render.SVG(payload), *view.output)
	case "html":
		return writeRenderedOutput(w, render.HTML(payload), *view.output)
	default:
		return writeConfigError(w, "unsupported --format: "+*view.format)
	}
	return ExitOK
}

func emitVisualResponse(w io.Writer, apiClient *client.Client, response map[string]any, metadata map[string]any, view renderFlags) int {
	format := strings.ToLower(strings.TrimSpace(*view.format))
	if format == "" || format == "json" {
		return emitResponse(w, response, metadata, view)
	}
	payload := mergePayload(response, metadata)
	if schedules, err := apiClient.Get(context.Background(), "/api/laps/schedules", url.Values{"pageSize": []string{"500"}}); err == nil {
		payload = mergeVisualSchedules(payload, schedules)
	}
	return emitResponse(w, payload, nil, view)
}

func writeRenderedOutput(w io.Writer, content string, output string) int {
	if strings.TrimSpace(output) == "" {
		fmt.Fprint(w, content)
		return ExitOK
	}
	if err := os.WriteFile(output, []byte(content), 0o644); err != nil {
		writeError(w, outputPayload{
			Success: false,
			Error: &client.ErrorPayload{
				Code:    "CONFIG_ERROR",
				Message: err.Error(),
			},
		})
		return ExitUsage
	}
	fmt.Fprintf(w, "wrote %s\n", output)
	return ExitOK
}

func mergePayload(response map[string]any, metadata map[string]any) map[string]any {
	payload := make(map[string]any, len(response)+len(metadata))
	for key, value := range response {
		payload[key] = value
	}
	for key, value := range metadata {
		if key == "orderIds" {
			if ids, ok := value.([]string); ok && len(ids) == 0 {
				continue
			}
		}
		payload[key] = value
	}
	return payload
}

func mergeVisualSchedules(payload map[string]any, schedules map[string]any) map[string]any {
	records, ok := schedules["records"].([]any)
	if !ok || len(records) == 0 {
		return payload
	}
	merged := make(map[string]any, len(payload)+1)
	for key, value := range payload {
		merged[key] = value
	}
	existing, _ := merged["records"].([]any)
	all := append([]any{}, existing...)
	all = append(all, records...)
	merged["records"] = uniqueVisualRecords(all)
	return merged
}

func uniqueVisualRecords(records []any) []any {
	seen := map[string]bool{}
	unique := make([]any, 0, len(records))
	for _, record := range records {
		m, ok := record.(map[string]any)
		if !ok {
			unique = append(unique, record)
			continue
		}
		key := visualRecordKey(m)
		if key != "" && seen[key] {
			continue
		}
		if key != "" {
			seen[key] = true
		}
		unique = append(unique, record)
	}
	return unique
}

func visualRecordKey(record map[string]any) string {
	parts := []string{
		fmt.Sprint(record["id"]),
		fmt.Sprint(record["orderId"]),
		fmt.Sprint(record["teamId"]),
		fmt.Sprint(record["teamName"]),
		fmt.Sprint(record["startDate"]),
		fmt.Sprint(record["endDate"]),
	}
	return strings.Join(parts, "|")
}

func writeJSON(w io.Writer, payload outputPayload) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}
