package commands

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"production-scheduling-cli/internal/client"
)

type keyValues []string

func (s *keyValues) String() string { return strings.Join(*s, ",") }
func (s *keyValues) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--set value cannot be empty")
	}
	*s = append(*s, value)
	return nil
}

type crudSpec struct {
	command  string
	path     string
	queryKey string
	fields   []string
}

func runCRUD(spec crudSpec, args []string, stdout io.Writer) int {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprintf(stdout, "Usage:\n  laps-cli %s list|get|create|update|delete [flags]\n", spec.command)
		return map[bool]int{true: ExitOK, false: ExitUsage}[len(args) > 0]
	}
	action := args[0]
	command := spec.command + " " + action
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	id := fs.String("id", "", "record ID")
	filePath := fs.String("file", "", "JSON file or - for stdin")
	query := fs.String("query", "", "search text")
	status := fs.String("status", "", "status filter")
	page := fs.Int("page", 1, "page number")
	limit := fs.Int("limit", 100, "page size")
	pageToken := fs.String("page-token", "", "pagination token")
	yes := fs.Bool("yes", false, "confirm destructive deletion")
	var sets keyValues
	fs.Var(&sets, "set", "field=value; repeat for multiple fields")
	directFields := make(map[string]*string, len(spec.fields))
	for _, field := range spec.fields {
		directFields[field] = fs.String(field, "", "business field")
	}
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	for field, value := range directFields {
		if *value != "" {
			sets = append(sets, field+"="+*value)
		}
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	path := spec.path
	var response map[string]any
	var err error
	switch action {
	case "list":
		values := url.Values{}
		if *query != "" {
			values.Set(spec.queryKey, *query)
		}
		if *status != "" {
			values.Set("status", *status)
		}
		values.Set("page", strconv.Itoa(*page))
		values.Set("pageSize", strconv.Itoa(*limit))
		if *pageToken != "" {
			values.Set("pageToken", *pageToken)
		}
		response, err = apiClient.Get(context.Background(), path, values)
	case "get":
		if strings.TrimSpace(*id) == "" {
			return writeConfigError(stdout, "--id is required")
		}
		response, err = apiClient.Get(context.Background(), path+"/"+url.PathEscape(*id), nil)
	case "create":
		payload, code := readPayload(*filePath, sets, os.Stdin, stdout)
		if code != ExitOK {
			return code
		}
		response, err = apiClient.Post(context.Background(), path, payload)
	case "update":
		if strings.TrimSpace(*id) == "" {
			return writeConfigError(stdout, "--id is required")
		}
		payload, code := readPayload(*filePath, sets, os.Stdin, stdout)
		if code != ExitOK {
			return code
		}
		response, err = apiClient.Patch(context.Background(), path+"/"+url.PathEscape(*id), payload)
	case "delete":
		if strings.TrimSpace(*id) == "" {
			return writeConfigError(stdout, "--id is required")
		}
		if !*yes {
			return writeConfigError(stdout, "delete requires --yes")
		}
		response, err = apiClient.Delete(context.Background(), path+"/"+url.PathEscape(*id))
	default:
		return writeConfigError(stdout, "unknown "+spec.command+" action: "+action)
	}
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}

func readPayload(filePath string, sets []string, stdin io.Reader, stdout io.Writer) (map[string]any, int) {
	if filePath != "" && len(sets) > 0 {
		return nil, writeConfigError(stdout, "--file and --set cannot be used together")
	}
	if filePath == "" && len(sets) == 0 {
		return nil, writeConfigError(stdout, "provide --file or at least one --set")
	}
	payload := map[string]any{}
	if filePath != "" {
		var raw []byte
		var err error
		if filePath == "-" {
			raw, err = io.ReadAll(stdin)
		} else {
			raw, err = os.ReadFile(filePath)
		}
		if err != nil {
			return nil, writeConfigError(stdout, "read input: "+err.Error())
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, writeConfigError(stdout, "parse JSON: "+err.Error())
		}
		return payload, ExitOK
	}
	for _, item := range sets {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, writeConfigError(stdout, "--set must use field=value")
		}
		payload[toCamelCase(strings.TrimSpace(key))] = parseScalar(value)
	}
	return payload, ExitOK
}

func toCamelCase(value string) string {
	parts := strings.Split(value, "-")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func parseScalar(value string) any {
	trimmed := strings.TrimSpace(value)
	var parsed any
	if json.Unmarshal([]byte(trimmed), &parsed) == nil {
		return parsed
	}
	return value
}

func runOrdersExtended(args []string, stdout io.Writer) int {
	if len(args) > 0 && args[0] == "import" {
		return runImport("orders", "/api/laps/orders/imports", args[1:], stdout)
	}
	if len(args) > 0 && args[0] == "export" {
		return runOrdersExport(args[1:], stdout)
	}
	return runCRUD(crudSpec{
		command: "orders", path: "/api/laps/orders", queryKey: "query",
		fields: []string{"sequence-no", "style-no", "style-desc", "style-type", "category-id", "customer-name", "order-qty", "delivery-date", "sam-value", "factory", "order-status"},
	}, args, stdout)
}

func runOrdersExport(args []string, stdout io.Writer) int {
	command := "orders export"
	if wantsHelp(args) {
		fmt.Fprintln(stdout, "Usage:\n  laps-cli orders export --output FILE [--format csv|json]")
		return ExitOK
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	format := fs.String("format", "csv", "csv | json")
	output := fs.String("output", "", "output file")
	if code := parseNoArgs(fs, args, stdout, command); code != ExitOK {
		return code
	}
	if *output == "" {
		return writeConfigError(stdout, "--output is required")
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	response := map[string]any{"success": true, "records": []any{}}
	allRecords := make([]any, 0)
	pageToken := ""
	for {
		values := url.Values{"status": {"all"}, "pageSize": {"500"}}
		if pageToken != "" {
			values.Set("pageToken", pageToken)
		}
		page, err := apiClient.Get(context.Background(), "/api/laps/orders", values)
		if err != nil {
			return writeAPIError(stdout, err, command, nil, nil)
		}
		if records, ok := page["records"].([]any); ok {
			allRecords = append(allRecords, records...)
		}
		hasMore, _ := page["hasMore"].(bool)
		pageToken = stringValue(page["pageToken"])
		if !hasMore || pageToken == "" {
			break
		}
	}
	response["records"] = allRecords
	response["total"] = len(allRecords)
	var raw []byte
	var err error
	if *format == "json" {
		raw, err = json.MarshalIndent(response, "", "  ")
	} else if *format == "csv" {
		var buffer strings.Builder
		writer := csv.NewWriter(&buffer)
		_ = writer.Write([]string{"id", "sequenceNo", "styleNo", "styleDesc", "customerName", "orderQty", "remainingQty", "deliveryDate", "status"})
		for _, item := range asRecords(response["records"]) {
			_ = writer.Write([]string{stringValue(item["id"]), stringValue(item["sequenceNo"]), stringValue(item["styleNo"]), stringValue(item["styleDesc"]), stringValue(item["customerName"]), stringValue(item["orderQty"]), stringValue(item["remainingQty"]), stringValue(item["deliveryDate"]), stringValue(item["status"])})
		}
		writer.Flush()
		raw = []byte(buffer.String())
		err = writer.Error()
	} else {
		return writeConfigError(stdout, "invalid --format: "+*format)
	}
	if err != nil {
		return writeConfigError(stdout, err.Error())
	}
	if err := os.WriteFile(*output, raw, 0o644); err != nil {
		return writeConfigError(stdout, err.Error())
	}
	writeJSONMap(stdout, map[string]any{"success": true, "command": command, "output": *output, "format": *format, "bytes": len(raw)})
	return ExitOK
}

func runCalendars(args []string, stdout io.Writer) int {
	if len(args) > 0 && args[0] == "bind" {
		if wantsHelp(args[1:]) {
			fmt.Fprintln(stdout, "Usage:\n  laps-cli calendars bind --calendar-id ID --team-id ID")
			return ExitOK
		}
		command := "calendars bind"
		fs := flag.NewFlagSet(command, flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		common := addCommonFlags(fs)
		calendarID := fs.String("calendar-id", "", "calendar ID")
		teamID := fs.String("team-id", "", "team ID")
		if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
			return code
		}
		if *calendarID == "" || *teamID == "" {
			return writeConfigError(stdout, "--calendar-id and --team-id are required")
		}
		apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
		if !ok {
			return ExitUsage
		}
		response, err := apiClient.Put(context.Background(), "/api/laps/calendars/"+url.PathEscape(*calendarID)+"/teams/"+url.PathEscape(*teamID), map[string]any{})
		if err != nil {
			return writeAPIError(stdout, err, command, nil, nil)
		}
		writeSuccess(stdout, response, map[string]any{"command": command})
		return ExitOK
	}
	return runCRUD(crudSpec{command: "calendars", path: "/api/laps/calendars", queryKey: "q", fields: []string{"calendar-code", "calendar-name", "factory", "production-line"}}, args, stdout)
}

func runImport(domain string, path string, args []string, stdout io.Writer) int {
	prefix := domain
	if domain == "orders" {
		prefix = "orders import"
	}
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprintf(stdout, "Usage:\n  laps-cli %s template|preview|apply [flags]\n", prefix)
		if len(args) == 0 {
			return ExitUsage
		}
		return ExitOK
	}
	action := args[0]
	command := prefix + " " + action
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	filePath := fs.String("file", "", "JSON or Excel file")
	output := fs.String("output", "", "download destination")
	mode := fs.String("mode", "create", "create | upsert")
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	if action == "template" {
		if *output == "" {
			return writeConfigError(stdout, "--output is required")
		}
		raw, _, err := apiClient.Download(context.Background(), path+"/template/file")
		if err != nil {
			return writeAPIError(stdout, err, command, nil, nil)
		}
		if err := os.WriteFile(*output, raw, 0o644); err != nil {
			return writeConfigError(stdout, err.Error())
		}
		writeJSONMap(stdout, map[string]any{"success": true, "command": command, "output": *output, "bytes": len(raw)})
		return ExitOK
	}
	if action != "preview" && action != "apply" {
		return writeConfigError(stdout, "unknown import action: "+action)
	}
	if *filePath == "" {
		return writeConfigError(stdout, "--file is required")
	}
	var response map[string]any
	var err error
	if strings.EqualFold(filepath.Ext(*filePath), ".xlsx") || strings.EqualFold(filepath.Ext(*filePath), ".xls") {
		response, err = apiClient.Upload(context.Background(), path+"/file/"+action, *filePath, map[string]string{"mode": *mode})
	} else {
		payload, code := readPayload(*filePath, nil, os.Stdin, stdout)
		if code != ExitOK {
			return code
		}
		if domain == "orders" {
			payload["mode"] = *mode
		}
		response, err = apiClient.Post(context.Background(), path+"/"+action, payload)
	}
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command, "file": *filePath})
	return ExitOK
}

func runMaterialImport(args []string, stdout io.Writer) int {
	if len(args) > 0 && args[0] == "history" {
		if wantsHelp(args[1:]) {
			fmt.Fprintln(stdout, "Usage:\n  laps-cli material-import history [--limit N]")
			return ExitOK
		}
		fs := flag.NewFlagSet("material-import history", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		common := addCommonFlags(fs)
		limit := fs.Int("limit", 50, "record limit")
		if code := parseNoArgs(fs, args[1:], stdout, "material-import history"); code != ExitOK {
			return code
		}
		return runGet(stdout, common, "material-import history", "/api/laps/material-imports", url.Values{"limit": {strconv.Itoa(*limit)}})
	}
	return runImport("material-import", "/api/laps/material-imports", args, stdout)
}

func runReadiness(args []string, stdout io.Writer) int {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprintln(stdout, "Usage:\n  laps-cli readiness status|latest|get|schema|analyze|external [flags]")
		if len(args) == 0 {
			return ExitUsage
		}
		return ExitOK
	}
	action := args[0]
	if action == "external" && len(args) > 1 && args[1] == "import" {
		args = append([]string{"external"}, args[2:]...)
	}
	command := "readiness " + action
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	analysisID := fs.String("analysis-id", "", "analysis ID")
	filePath := fs.String("file", "", "request JSON file or -")
	var orderIDs stringList
	fs.Var(&orderIDs, "order-id", "order ID; repeat")
	source := fs.String("source", "auto", "auto | builtin | external")
	persist := fs.Bool("persist", true, "persist analysis")
	full := fs.Bool("full", false, "include the full latest readiness result")
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	var response map[string]any
	var err error
	switch action {
	case "status", "schema":
		response, err = apiClient.Get(context.Background(), "/api/laps/readiness/"+action, nil)
	case "latest":
		response, err = apiClient.Get(context.Background(), "/api/laps/readiness/latest", url.Values{"full": {strconv.FormatBool(*full)}})
	case "get":
		if *analysisID == "" {
			return writeConfigError(stdout, "--analysis-id is required")
		}
		response, err = apiClient.Get(context.Background(), "/api/laps/readiness/analyses/"+url.PathEscape(*analysisID), nil)
	case "analyze":
		payload := map[string]any{"orderIds": []string(orderIDs), "source": *source, "persist": *persist}
		if *filePath != "" {
			var code int
			payload, code = readPayload(*filePath, nil, os.Stdin, stdout)
			if code != ExitOK {
				return code
			}
		}
		response, err = apiClient.Post(context.Background(), "/api/laps/readiness/analyze", payload)
	case "external":
		payload, code := readPayload(*filePath, nil, os.Stdin, stdout)
		if code != ExitOK {
			return code
		}
		response, err = apiClient.Post(context.Background(), "/api/laps/readiness/external-results", payload)
	default:
		return writeConfigError(stdout, "unknown readiness action: "+action)
	}
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}

func runResources(args []string, stdout io.Writer) int {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprintln(stdout, "Usage:\n  laps-cli resources list|get|apply|batch-settings|delete-factory|delete-team [flags]")
		if len(args) == 0 {
			return ExitUsage
		}
		return ExitOK
	}
	if args[0] == "batch-settings" {
		return runFactoryBatchSettings(args[1:], stdout)
	}
	action := args[0]
	command := "resources " + action
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	factoryID := fs.String("factory-id", "", "factory ID")
	teamID := fs.String("team-id", "", "team ID")
	filePath := fs.String("file", "", "factory tree JSON file or -")
	includeInactive := fs.Bool("include-inactive", true, "include inactive resources")
	yes := fs.Bool("yes", false, "confirm deletion")
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	var response map[string]any
	var err error
	switch action {
	case "list":
		response, err = apiClient.Get(context.Background(), "/api/laps/resources", url.Values{"includeInactive": {strconv.FormatBool(*includeInactive)}})
	case "get":
		if *factoryID == "" {
			return writeConfigError(stdout, "--factory-id is required")
		}
		response, err = apiClient.Get(context.Background(), "/api/laps/resources/factories/"+url.PathEscape(*factoryID), nil)
	case "apply":
		payload, code := readPayload(*filePath, nil, os.Stdin, stdout)
		if code != ExitOK {
			return code
		}
		if *factoryID == "" {
			response, err = apiClient.Post(context.Background(), "/api/laps/resources/factories", payload)
		} else {
			response, err = apiClient.Put(context.Background(), "/api/laps/resources/factories/"+url.PathEscape(*factoryID), payload)
		}
	case "delete-factory":
		if *factoryID == "" || !*yes {
			return writeConfigError(stdout, "delete-factory requires --factory-id and --yes")
		}
		response, err = apiClient.Delete(context.Background(), "/api/laps/resources/factories/"+url.PathEscape(*factoryID))
	case "delete-team":
		if *teamID == "" || !*yes {
			return writeConfigError(stdout, "delete-team requires --team-id and --yes")
		}
		response, err = apiClient.Delete(context.Background(), "/api/laps/resources/teams/"+url.PathEscape(*teamID))
	default:
		return writeConfigError(stdout, "unknown resources action: "+action)
	}
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}

func runFactoryBatchSettings(args []string, stdout io.Writer) int {
	command := "resources batch-settings"
	if wantsHelp(args) {
		fmt.Fprintln(stdout, "Usage:\n  laps-cli resources batch-settings --factory-id ID [--factory-id ID ...] [--enabled true|false] [--ownership-type owned|outsourced] [--is-headquarters true|false]\n  laps-cli resources batch-settings --file JSON")
		return ExitOK
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	var factoryIDs stringList
	fs.Var(&factoryIDs, "factory-id", "factory ID; repeat")
	filePath := fs.String("file", "", "batch settings JSON file or -")
	enabled := fs.String("enabled", "", "true | false")
	ownershipType := fs.String("ownership-type", "", "owned | outsourced")
	isHeadquarters := fs.String("is-headquarters", "", "true | false")
	if code := parseNoArgs(fs, args, stdout, command); code != ExitOK {
		return code
	}
	hasFlags := len(factoryIDs) > 0 || *enabled != "" || *ownershipType != "" || *isHeadquarters != ""
	if *filePath != "" && hasFlags {
		return writeConfigError(stdout, "--file cannot be used with batch-settings flags")
	}
	var payload map[string]any
	if *filePath != "" {
		var code int
		payload, code = readPayload(*filePath, nil, os.Stdin, stdout)
		if code != ExitOK {
			return code
		}
	} else {
		if len(factoryIDs) == 0 {
			return writeConfigError(stdout, "at least one --factory-id is required")
		}
		if *enabled == "" && *ownershipType == "" && *isHeadquarters == "" {
			return writeConfigError(stdout, "provide --enabled, --ownership-type, or --is-headquarters")
		}
		payload = map[string]any{"factoryIds": []string(factoryIDs)}
		if *enabled != "" {
			value, err := strconv.ParseBool(*enabled)
			if err != nil {
				return writeConfigError(stdout, "--enabled must be true or false")
			}
			payload["enabled"] = value
		}
		if *ownershipType != "" {
			if *ownershipType != "owned" && *ownershipType != "outsourced" {
				return writeConfigError(stdout, "--ownership-type must be owned or outsourced")
			}
			payload["ownershipType"] = *ownershipType
		}
		if *isHeadquarters != "" {
			value, err := strconv.ParseBool(*isHeadquarters)
			if err != nil {
				return writeConfigError(stdout, "--is-headquarters must be true or false")
			}
			payload["isHeadquarters"] = value
		}
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	response, err := apiClient.Patch(context.Background(), "/api/laps/resources/factories/batch-settings", payload)
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}

func runCapacityDomain(args []string, stdout io.Writer) int {
	action := args[0]
	command := "capacity " + action
	if wantsHelp(args[1:]) {
		fmt.Fprintf(stdout, "Usage:\n  laps-cli %s [flags]\n", command)
		return ExitOK
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	resource := fs.String("resource", "plans", "capacity resource")
	id := fs.String("id", "", "record ID")
	factoryID := fs.String("factory-id", "", "factory ID")
	filePath := fs.String("file", "", "JSON file or -")
	yes := fs.Bool("yes", false, "confirm deletion")
	var sets keyValues
	fs.Var(&sets, "set", "field=value; repeat")
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	var response map[string]any
	var err error
	if action == "profiles" {
		if *factoryID == "" {
			return writeConfigError(stdout, "--factory-id is required")
		}
		if *filePath == "" {
			response, err = apiClient.Get(context.Background(), "/api/laps/capacity/profiles/by-factory/"+url.PathEscape(*factoryID), nil)
		} else {
			payload, code := readPayload(*filePath, sets, os.Stdin, stdout)
			if code != ExitOK {
				return code
			}
			response, err = apiClient.Put(context.Background(), "/api/laps/capacity/profiles/by-factory/"+url.PathEscape(*factoryID), payload)
		}
	} else {
		if !validCapacityResource(*resource) {
			return writeConfigError(stdout, "invalid capacity resource: "+*resource)
		}
		basePath := "/api/laps/capacity/" + *resource
		switch action {
		case "get":
			if *id == "" {
				return writeConfigError(stdout, "--id is required")
			}
			response, err = apiClient.Get(context.Background(), basePath+"/"+url.PathEscape(*id), nil)
		case "create":
			payload, code := readPayload(*filePath, sets, os.Stdin, stdout)
			if code != ExitOK {
				return code
			}
			response, err = apiClient.Post(context.Background(), basePath, payload)
		case "update":
			if *id == "" {
				return writeConfigError(stdout, "--id is required")
			}
			payload, code := readPayload(*filePath, sets, os.Stdin, stdout)
			if code != ExitOK {
				return code
			}
			response, err = apiClient.Patch(context.Background(), basePath+"/"+url.PathEscape(*id), payload)
		case "delete":
			if *id == "" || !*yes {
				return writeConfigError(stdout, "delete requires --id and --yes")
			}
			response, err = apiClient.Delete(context.Background(), basePath+"/"+url.PathEscape(*id))
		default:
			return writeConfigError(stdout, "unknown capacity action: "+action)
		}
	}
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}

func runCapacityProfiles(args []string, stdout io.Writer) int {
	action := args[0]
	command := "capacity profiles " + action
	if wantsHelp(args[1:]) {
		fmt.Fprintf(stdout, "Usage:\n  laps-cli %s --factory-id ID [--file JSON]\n", command)
		return ExitOK
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	factoryID := fs.String("factory-id", "", "factory ID")
	filePath := fs.String("file", "", "profile batch JSON file or -")
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	if *factoryID == "" {
		return writeConfigError(stdout, "--factory-id is required")
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	path := "/api/laps/capacity/profiles/by-factory/" + url.PathEscape(*factoryID)
	var response map[string]any
	var err error
	if action == "get" {
		if *filePath != "" {
			return writeConfigError(stdout, "capacity profiles get does not accept --file")
		}
		response, err = apiClient.Get(context.Background(), path, nil)
	} else {
		payload, code := readPayload(*filePath, nil, os.Stdin, stdout)
		if code != ExitOK {
			return code
		}
		response, err = apiClient.Put(context.Background(), path, payload)
	}
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command, "factoryId": *factoryID})
	return ExitOK
}

func runCapacityImport(args []string, stdout io.Writer) int {
	action := args[0]
	command := "capacity import " + action
	if wantsHelp(args[1:]) {
		fmt.Fprintf(stdout, "Usage:\n  laps-cli %s [flags]\n", command)
		return ExitOK
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	filePath := fs.String("file", "", "normalized JSON or Excel file")
	planCode := fs.String("plan-code", "", "capacity plan code for Excel")
	planName := fs.String("plan-name", "", "capacity plan name for Excel")
	periodStart := fs.String("period-start", "", "YYYY-MM-DD for Excel")
	periodEnd := fs.String("period-end", "", "YYYY-MM-DD for Excel")
	version := fs.Int("version", 1, "plan version")
	output := fs.String("output", "", "template download destination")
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	if action == "template" {
		if *output == "" {
			return writeConfigError(stdout, "--output is required")
		}
		apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
		if !ok {
			return ExitUsage
		}
		raw, _, err := apiClient.Download(context.Background(), "/api/laps/capacity/imports/template/file")
		if err != nil {
			return writeAPIError(stdout, err, command, nil, nil)
		}
		if err := os.WriteFile(*output, raw, 0o644); err != nil {
			return writeConfigError(stdout, err.Error())
		}
		writeJSONMap(stdout, map[string]any{"success": true, "command": command, "output": *output, "bytes": len(raw)})
		return ExitOK
	}
	if action != "preview" && action != "apply" {
		return writeConfigError(stdout, "unknown capacity import action: "+action)
	}
	if *filePath == "" {
		return writeConfigError(stdout, "--file is required")
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	var response map[string]any
	var err error
	if strings.EqualFold(filepath.Ext(*filePath), ".xlsx") || strings.EqualFold(filepath.Ext(*filePath), ".xls") {
		if *planCode == "" || *planName == "" || *periodStart == "" || *periodEnd == "" {
			return writeConfigError(stdout, "Excel import requires --plan-code --plan-name --period-start --period-end")
		}
		plan, _ := json.Marshal(map[string]any{"code": *planCode, "name": *planName, "periodStart": *periodStart, "periodEnd": *periodEnd, "version": *version})
		response, err = apiClient.Upload(context.Background(), "/api/laps/capacity/imports/file/"+action, *filePath, map[string]string{"plan": string(plan)})
	} else {
		payload, code := readPayload(*filePath, nil, os.Stdin, stdout)
		if code != ExitOK {
			return code
		}
		response, err = apiClient.Post(context.Background(), "/api/laps/capacity/imports/"+action, payload)
	}
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command, "file": *filePath})
	return ExitOK
}

func asRecords(value any) []map[string]any {
	items, _ := value.([]any)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if record, ok := item.(map[string]any); ok {
			result = append(result, record)
		}
	}
	return result
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func writeDomainUsage(w io.Writer) {
	fmt.Fprintln(w, "  laps-cli materials summary|list|get|create|update|delete")
	fmt.Fprintln(w, "  laps-cli boms list|get|create|update|delete")
	fmt.Fprintln(w, "  laps-cli material-import template|history|preview|apply")
	fmt.Fprintln(w, "  laps-cli readiness status|latest|get|schema|analyze")
	fmt.Fprintln(w, "  laps-cli readiness external import --file JSON")
	fmt.Fprintln(w, "  laps-cli resources list|get|apply|batch-settings|delete-factory|delete-team")
	fmt.Fprintln(w, "  laps-cli efficiencies|calendars|holidays list|get|create|update|delete")
	fmt.Fprintln(w, "  laps-cli calendars bind --calendar-id ID --team-id ID")
	fmt.Fprintln(w, "  laps-cli scheduling-policy list|get|create|update|delete|clone|validate|publish|runs")
}

func writeClientError(stdout io.Writer, err error, command string) int {
	return writeAPIError(stdout, client.AsAPIError(err), command, nil, nil)
}
