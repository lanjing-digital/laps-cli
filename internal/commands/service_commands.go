package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Finite mappings: no caller-supplied URL, HTTP verb or arbitrary request parameters.
type serviceCommand struct {
	path        string
	method      string
	id          bool
	body        bool
	confirm     bool
	destructive bool
	query       map[string]string
	category    string
}

var serviceCommands = map[string]serviceCommand{
	"execution usage preview":          {path: "/api/laps/execution/material-usage-preview", query: map[string]string{"target-type": "targetType", "target-id": "targetId", "output-quantity": "output"}},
	"personal profile get":             {path: "/api/laps/personal/profile"},
	"personal memories list":           {path: "/api/laps/personal/memories", query: map[string]string{"category": "category"}},
	"personal memories get":            {path: "/api/laps/personal/memories/{id}", id: true},
	"personal memories create":         {path: "/api/laps/personal/memories", method: http.MethodPost, body: true, confirm: true},
	"personal memories update":         {path: "/api/laps/personal/memories/{id}", id: true, method: http.MethodPatch, body: true, confirm: true},
	"personal memories delete":         {path: "/api/laps/personal/memories/{id}", id: true, method: http.MethodDelete, confirm: true, destructive: true, query: map[string]string{"version": "version"}},
	"personal memories clear":          {path: "/api/laps/personal/memories/clear", method: http.MethodPost, confirm: true, destructive: true},
	"personal preferences list":        {path: "/api/laps/personal/memories", category: "preference"},
	"personal preferences create":      {path: "/api/laps/personal/memories", category: "preference", method: http.MethodPost, body: true, confirm: true},
	"personal preferences update":      {path: "/api/laps/personal/memories/{id}", category: "preference", id: true, method: http.MethodPatch, body: true, confirm: true},
	"personal learning get":            {path: "/api/laps/personal/learning"},
	"personal learning set":            {path: "/api/laps/personal/learning", method: http.MethodPatch, confirm: true},
	"personal learning initialize":     {path: "/api/laps/personal/learning/initialize", method: http.MethodPost, confirm: true},
	"personal learning retry":          {path: "/api/laps/personal/learning/retry", method: http.MethodPost, confirm: true},
	"personal interests list":          {path: "/api/laps/personal/interests"},
	"personal interests replace":       {path: "/api/laps/personal/interests", method: http.MethodPatch, body: true, confirm: true},
	"personal workbench get":           {path: "/api/laps/personal/workbench"},
	"personal workbench facts":         {path: "/api/laps/personal/facts/{id}", id: true},
	"personal workbench guidance":      {path: "/api/laps/personal/guidance/{id}", id: true},
	"personal workbench refresh":       {path: "/api/laps/personal/workbench/refresh", method: http.MethodPost, confirm: true},
	"personal sync preview":            {path: "/api/laps/personal/sync/preview", method: http.MethodPost, body: true},
	"personal sync apply":              {path: "/api/laps/personal/sync/apply", method: http.MethodPost, body: true, confirm: true},
	"conversations list":               {path: "/api/laps/agent-conversations"},
	"conversations get":                {path: "/api/laps/agent-conversations/{id}", id: true},
	"procurement preview":              {path: "/api/laps/readiness/procurement-imports/preview", method: http.MethodPost, body: true},
	"procurement apply":                {path: "/api/laps/readiness/procurement-imports/apply", method: http.MethodPost, body: true, confirm: true},
	"planning status":                  {path: "/api/laps/production-planning/status"},
	"planning plans list":              {path: "/api/laps/production-planning/plans", query: map[string]string{"from": "from", "to": "to"}},
	"planning plans get":               {path: "/api/laps/production-planning/plans/{id}", id: true},
	"planning plans save":              {path: "/api/laps/production-planning/plans", method: http.MethodPost, body: true, confirm: true},
	"planning plans analyze":           {path: "/api/laps/production-planning/plans/{id}/analyze", id: true, method: http.MethodPost, body: true, confirm: true},
	"planning plans confirm":           {path: "/api/laps/production-planning/plans/{id}/confirm", id: true, method: http.MethodPost, body: true, confirm: true},
	"planning plans import-readiness":  {path: "/api/laps/production-planning/plans/import-readiness", method: http.MethodPost, body: true, confirm: true},
	"planning followups list":          {path: "/api/laps/production-planning/followups", query: map[string]string{"plan-id": "planId"}},
	"planning followups save":          {path: "/api/laps/production-planning/followups", method: http.MethodPost, body: true, confirm: true},
	"planning preparations options":    {path: "/api/laps/production-planning/preparations/options", query: map[string]string{"plan-line-id": "planLineId"}},
	"planning preparations apply":      {path: "/api/laps/production-planning/preparations", method: http.MethodPost, body: true, confirm: true},
	"warehouse warehouses list":        {path: "/api/laps/warehouse-inventory/warehouses"},
	"warehouse warehouses get":         {path: "/api/laps/warehouse-inventory/warehouses/{id}", id: true},
	"warehouse warehouses create":      {path: "/api/laps/warehouse-inventory/warehouses", method: http.MethodPost, body: true, confirm: true},
	"warehouse warehouses update":      {path: "/api/laps/warehouse-inventory/warehouses/{id}", id: true, method: http.MethodPatch, body: true, confirm: true},
	"warehouse warehouses assignments": {path: "/api/laps/warehouse-inventory/warehouses/{id}/assignments", id: true, method: http.MethodPost, body: true, confirm: true},
	"warehouse warehouses mode":        {path: "/api/laps/warehouse-inventory/warehouses/{id}/mode", id: true, method: http.MethodPost, body: true, confirm: true},
	"warehouse users list":             {path: "/api/laps/warehouse-inventory/users"},
	"warehouse inventory list":         {path: "/api/laps/warehouse-inventory/inventory", query: map[string]string{"warehouse-id": "warehouseId", "material-code": "materialCode", "batch-no": "batchNo", "stock-status": "stockStatus"}},
	"warehouse inventory adjust":       {path: "/api/laps/warehouse-inventory/inventory/direct-adjust", method: http.MethodPost, body: true, confirm: true},
	"warehouse inbounds list":          {path: "/api/laps/warehouse-inventory/inbounds", query: map[string]string{"warehouse-id": "warehouseId"}},
	"warehouse documents list":         {path: "/api/laps/warehouse-inventory/documents", query: map[string]string{"warehouse-id": "warehouseId"}},
	"warehouse documents create":       {path: "/api/laps/warehouse-inventory/documents", method: http.MethodPost, body: true, confirm: true},
	"warehouse documents post":         {path: "/api/laps/warehouse-inventory/documents/{id}/post", id: true, method: http.MethodPost, confirm: true},
	"warehouse documents void":         {path: "/api/laps/warehouse-inventory/documents/{id}/void", id: true, method: http.MethodPost, confirm: true, destructive: true},
	"warehouse connectors save":        {path: "/api/laps/warehouse-inventory/connectors", method: http.MethodPost, body: true, confirm: true},
	"warehouse sync preview":           {path: "/api/laps/warehouse-inventory/sync/preview", method: http.MethodPost, body: true},
	"warehouse sync apply":             {path: "/api/laps/warehouse-inventory/sync/apply", method: http.MethodPost, body: true, confirm: true},
	"warehouse sync history":           {path: "/api/laps/warehouse-inventory/sync/history"},
	"execution progress list":          {path: "/api/laps/execution/progress", query: map[string]string{"target-type": "targetType", "status": "status", "resource-id": "resourceId"}},
	"execution reports list":           {path: "/api/laps/execution/reports", query: map[string]string{"target-type": "targetType", "target-id": "targetId"}},
	"execution reports create":         {path: "/api/laps/execution/reports", method: http.MethodPost, body: true, confirm: true},
	"execution settings get":           {path: "/api/laps/execution/settings"},
	"execution settings update":        {path: "/api/laps/execution/settings", method: http.MethodPut, body: true, confirm: true},
	"execution bootstrap preview":      {path: "/api/laps/execution/historical-bootstrap/preview"},
	"execution bootstrap apply":        {path: "/api/laps/execution/historical-bootstrap/apply", method: http.MethodPost, body: true, confirm: true},
	"execution replans list":           {path: "/api/laps/execution/replan-cases"},
	"execution replans get":            {path: "/api/laps/execution/replan-cases/{id}", id: true},
	"execution replans preview":        {path: "/api/laps/execution/replan-cases/{id}/preview", id: true, method: http.MethodPost, body: true},
	"execution replans apply":          {path: "/api/laps/execution/replan-cases/{id}/apply", id: true, method: http.MethodPost, body: true, confirm: true},
	"execution replans dismiss":        {path: "/api/laps/execution/replan-cases/{id}/dismiss", id: true, method: http.MethodPost, confirm: true},
}

func serviceUsage(w io.Writer, prefix string) {
	keys := []string{}
	for key := range serviceCommands {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		spec := serviceCommands[key]
		fmt.Fprintf(w, "  laps-cli %s", key)
		if spec.id {
			fmt.Fprint(w, " --id ID")
		}
		if spec.body {
			fmt.Fprint(w, " --file JSON|- (or --set field=value)")
		}
		if spec.confirm {
			fmt.Fprint(w, " --confirm")
		}
		if spec.destructive {
			fmt.Fprint(w, " --yes")
		}
		if key == "personal sync apply" {
			fmt.Fprint(w, " --preview-token TOKEN")
		}
		if key == "personal learning set" {
			fmt.Fprint(w, " --enabled=true|false")
		}
		if key == "personal memories delete" {
			fmt.Fprint(w, " --version N")
		}
		fmt.Fprintln(w)
	}
}

func runServiceCommand(args []string, stdout io.Writer) int {
	end := 1
	for end < len(args) && !strings.HasPrefix(args[end], "-") {
		end++
	}
	command := strings.Join(args[:end], " ")
	spec, ok := serviceCommands[command]
	if !ok {
		serviceUsage(stdout, command)
		if wantsHelp(args[end:]) {
			return ExitOK
		}
		return writeConfigError(stdout, "unknown command: "+command)
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	id := fs.String("id", "", "exact object ID")
	confirm := fs.Bool("confirm", false, "confirm this write")
	yes := fs.Bool("yes", false, "confirm deletion, clear or void")
	file := ""
	var sets keyValues
	if spec.body {
		fs.StringVar(&file, "file", "", "JSON file, or - for stdin")
		fs.Var(&sets, "set", "field=value (repeat)")
	}
	values := map[string]*string{}
	for key := range spec.query {
		values[key] = fs.String(key, "", "query parameter")
	}
	token := ""
	if command == "personal sync apply" {
		fs.StringVar(&token, "preview-token", "", "token from sync preview")
	}
	enabled := ""
	if command == "personal learning set" {
		fs.StringVar(&enabled, "enabled", "", "true or false (required)")
	}
	if wantsHelp(args[end:]) {
		serviceUsage(stdout, command)
		// Defaults can contain environment credentials; help must never render those values.
		fs.VisitAll(func(f *flag.Flag) { fmt.Fprintf(stdout, "  --%s: %s\n", f.Name, f.Usage) })
		return ExitOK
	}
	if code := parseNoArgs(fs, args[end:], stdout, command); code != ExitOK {
		return code
	}
	if spec.confirm && !*confirm {
		return writeConfigError(stdout, "write requires --confirm")
	}
	if spec.destructive && !*yes {
		return writeConfigError(stdout, "destructive action requires --yes")
	}
	if spec.id && (strings.TrimSpace(*id) == "" || strings.ContainsAny(*id, "/\\?#") || *id == "." || *id == "..") {
		return writeConfigError(stdout, "--id must be an exact object ID")
	}
	var payload map[string]any
	if spec.body {
		var code int
		payload, code = readPayload(file, sets, os.Stdin, stdout)
		if code != ExitOK {
			return code
		}
		if payload == nil {
			return writeConfigError(stdout, "JSON input must be an object")
		}
	}
	if spec.category != "" && spec.body {
		if v, ok := payload["category"]; ok && v != spec.category {
			return writeConfigError(stdout, "preferences require category=preference")
		}
		payload["category"] = spec.category
	}
	if command == "personal memories update" || command == "personal preferences update" {
		version, ok := payload["version"].(float64)
		if !ok || version < 1 || version != float64(int(version)) {
			return writeConfigError(stdout, "update requires current version in JSON")
		}
	}
	if command == "personal memories delete" {
		n, err := strconv.Atoi(*values["version"])
		if err != nil || n < 1 {
			return writeConfigError(stdout, "delete requires --version N")
		}
	}
	if command == "personal sync apply" {
		if strings.TrimSpace(token) == "" {
			return writeConfigError(stdout, "sync apply requires --preview-token")
		}
		payload = map[string]any{"data": payload, "previewToken": token, "confirm": true}
	}
	if command == "personal learning set" {
		if enabled != "true" && enabled != "false" {
			return writeConfigError(stdout, "--enabled=true|false is required")
		}
		payload = map[string]any{"learningEnabled": enabled == "true"}
	}
	if command == "personal memories clear" {
		payload = map[string]any{"confirm": true}
	}
	api, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	path := strings.ReplaceAll(spec.path, "{id}", url.PathEscape(*id))
	query := url.Values{}
	for flagName, key := range spec.query {
		if *values[flagName] != "" {
			query.Set(key, *values[flagName])
		}
	}
	if spec.category != "" && !spec.body {
		query.Set("category", spec.category)
	}
	if spec.destructive && spec.method == http.MethodDelete {
		query.Set("yes", "true")
	}
	var response map[string]any
	var err error
	method := spec.method
	if method == "" {
		method = http.MethodGet
	}
	if method == http.MethodGet {
		response, err = api.Get(context.Background(), path, query)
	} else {
		if len(query) > 0 {
			path += "?" + query.Encode()
		}
		if payload == nil && method != http.MethodDelete {
			payload = map[string]any{}
		}
		response, err = api.JSON(context.Background(), method, path, payload)
	}
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}
