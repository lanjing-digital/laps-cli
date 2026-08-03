package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const capacityCalendarPath = "/api/laps/capacity-calendar"

func runCapacityCalendar(args []string, stdout io.Writer) int {
	if len(args) == 0 || wantsHelp(args) {
		writeCapacityCalendarUsage(stdout)
		if len(args) == 0 {
			return ExitUsage
		}
		return ExitOK
	}

	switch args[0] {
	case "days", "category-days", "history":
		return runCapacityCalendarRead(args, stdout)
	case "range", "category-range":
		return runCapacityCalendarRange(args, stdout)
	case "category-import":
		return runCategoryCapacityCalendarImport(args[1:], stdout)
	default:
		return writeConfigError(stdout, "unknown capacity calendar action: "+args[0])
	}
}

func runCapacityCalendarRead(args []string, stdout io.Writer) int {
	action := args[0]
	command := "capacity calendar " + action
	if wantsHelp(args[1:]) {
		fmt.Fprintf(stdout, "Usage:\n  laps-cli %s [flags]\n", command)
		return ExitOK
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	resourceID := fs.String("resource-id", "", "resource ID")
	categoryID := fs.String("category-id", "", "category ID")
	startDate := fs.String("start-date", "", "YYYY-MM-DD")
	endDate := fs.String("end-date", "", "YYYY-MM-DD")
	limit := fs.Int("limit", 200, "history record limit")
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	values := url.Values{}
	if *resourceID != "" {
		values.Set("resourceId", *resourceID)
	}
	if *categoryID != "" {
		values.Set("categoryId", *categoryID)
	}
	if *startDate != "" {
		values.Set("startDate", *startDate)
	}
	if *endDate != "" {
		values.Set("endDate", *endDate)
	}
	path := capacityCalendarPath + "/" + action
	if action == "history" {
		path = capacityCalendarPath + "/history"
		values = url.Values{"limit": {strconv.Itoa(*limit)}}
	}
	response, err := apiClient.Get(context.Background(), path, values)
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}

func runCapacityCalendarRange(args []string, stdout io.Writer) int {
	if len(args) < 2 || wantsHelp(args[1:]) {
		writeCapacityCalendarUsage(stdout)
		return map[bool]int{true: ExitOK, false: ExitUsage}[wantsHelp(args[1:])]
	}
	domain := args[0]
	action := args[1]
	if action != "set" && action != "reset" {
		return writeConfigError(stdout, "capacity calendar "+domain+" requires set or reset")
	}
	command := "capacity calendar " + domain + " " + action
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	resourceID := fs.String("resource-id", "", "resource ID")
	categoryID := fs.String("category-id", "", "category ID")
	startDate := fs.String("start-date", "", "YYYY-MM-DD")
	endDate := fs.String("end-date", "", "YYYY-MM-DD")
	dailyOutput := fs.Float64("daily-output", -1, "daily output; required for set")
	reason := fs.String("reason", "", "adjustment reason")
	if code := parseNoArgs(fs, args[2:], stdout, command); code != ExitOK {
		return code
	}
	if *resourceID == "" || *startDate == "" || *endDate == "" {
		return writeConfigError(stdout, "--resource-id --start-date and --end-date are required")
	}
	if domain == "category-range" && *categoryID == "" {
		return writeConfigError(stdout, "--category-id is required for category-range")
	}
	if action == "set" && *dailyOutput < 0 {
		return writeConfigError(stdout, "--daily-output must be zero or greater")
	}
	payload := map[string]any{"resourceId": *resourceID, "startDate": *startDate, "endDate": *endDate}
	if domain == "category-range" {
		payload["categoryId"] = *categoryID
	}
	if action == "set" {
		payload["dailyOutput"] = *dailyOutput
		if *reason != "" {
			payload["reason"] = *reason
		}
	}
	path := capacityCalendarPath + "/" + domain
	if action == "reset" {
		path += "/reset"
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	response, err := apiClient.Post(context.Background(), path, payload)
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}

func runCategoryCapacityCalendarImport(args []string, stdout io.Writer) int {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprintln(stdout, "Usage:\n  laps-cli capacity calendar category-import template|preview|apply [--file JSON|XLSX]")
		if len(args) == 0 {
			return ExitUsage
		}
		return ExitOK
	}
	action := args[0]
	command := "capacity calendar category-import " + action
	if wantsHelp(args[1:]) {
		fmt.Fprintf(stdout, "Usage:\n  laps-cli %s [flags]\n", command)
		return ExitOK
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	filePath := fs.String("file", "", "JSON, XLSX, or XLS file; - reads JSON from stdin")
	output := fs.String("output", "", "template download destination")
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
		raw, _, err := apiClient.Download(context.Background(), capacityCalendarPath+"/category-imports/template/file")
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
		return writeConfigError(stdout, "unknown category-import action: "+action)
	}
	if *filePath == "" {
		return writeConfigError(stdout, "--file is required")
	}
	var response map[string]any
	var err error
	if isSpreadsheet(*filePath) {
		response, err = apiClient.Upload(context.Background(), capacityCalendarPath+"/category-imports/file/"+action, *filePath, nil)
	} else {
		payload, code := readPayload(*filePath, nil, os.Stdin, stdout)
		if code != ExitOK {
			return code
		}
		response, err = apiClient.Post(context.Background(), capacityCalendarPath+"/category-imports/"+action, payload)
	}
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command, "file": *filePath})
	return ExitOK
}

func isSpreadsheet(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".xlsx" || extension == ".xls"
}

func writeCapacityCalendarUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli capacity calendar days [--resource-id ID --start-date YYYY-MM-DD --end-date YYYY-MM-DD]")
	fmt.Fprintln(w, "  laps-cli capacity calendar range set|reset --resource-id ID --start-date DATE --end-date DATE [--daily-output N --reason TEXT]")
	fmt.Fprintln(w, "  laps-cli capacity calendar category-days [--resource-id ID --category-id ID --start-date DATE --end-date DATE]")
	fmt.Fprintln(w, "  laps-cli capacity calendar category-range set|reset --resource-id ID --category-id ID --start-date DATE --end-date DATE [--daily-output N --reason TEXT]")
	fmt.Fprintln(w, "  laps-cli capacity calendar category-import template|preview|apply [--file JSON|XLSX]")
	fmt.Fprintln(w, "  laps-cli capacity calendar history [--limit N]")
}
