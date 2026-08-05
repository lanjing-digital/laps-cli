package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"
)

func runSchedulingPolicy(args []string, stdout io.Writer) int {
	if len(args) == 0 || wantsHelp(args) {
		writeSchedulingPolicyUsage(stdout)
		if len(args) == 0 {
			return ExitUsage
		}
		return ExitOK
	}
	if args[0] == "runs" {
		return runSchedulingPolicyRuns(args[1:], stdout)
	}
	if args[0] == "clone" || args[0] == "validate" || args[0] == "publish" {
		return runSchedulingPolicyAction(args, stdout)
	}
	return runCRUD(crudSpec{command: "scheduling-policy", path: "/api/laps/scheduling-policies", queryKey: "status"}, args, stdout)
}

func runSchedulingPolicyAction(args []string, stdout io.Writer) int {
	action := args[0]
	command := "scheduling-policy " + action
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	id := fs.String("id", "", "policy ID")
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	if strings.TrimSpace(*id) == "" {
		return writeConfigError(stdout, "--id is required")
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	response, err := apiClient.Post(context.Background(), "/api/laps/scheduling-policies/"+url.PathEscape(*id)+"/"+action, map[string]any{})
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command, "id": *id})
	return ExitOK
}

func runSchedulingPolicyRuns(args []string, stdout io.Writer) int {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprintln(stdout, "Usage:\n  laps-cli scheduling-policy runs list [--mode MODE] [--status STATUS] [--solver SOLVER] [--from ISO] [--to ISO] [--limit N] [--page-token TOKEN]\n  laps-cli scheduling-policy runs get --run-id ID")
		if len(args) == 0 {
			return ExitUsage
		}
		return ExitOK
	}
	command := "scheduling-policy runs " + args[0]
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	common := addCommonFlags(fs)
	runID := fs.String("run-id", "", "optimizer run ID")
	mode := fs.String("mode", "", "solver mode")
	status := fs.String("status", "", "run status")
	solver := fs.String("solver", "", "selected solver")
	from := fs.String("from", "", "ISO timestamp")
	to := fs.String("to", "", "ISO timestamp")
	limit := fs.Int("limit", 50, "maximum records, up to 200")
	pageToken := fs.String("page-token", "", "next page token")
	if code := parseNoArgs(fs, args[1:], stdout, command); code != ExitOK {
		return code
	}
	apiClient, ok := newAPIClient(common, stdout, command, nil, nil)
	if !ok {
		return ExitUsage
	}
	var response map[string]any
	var err error
	switch args[0] {
	case "list":
		values := url.Values{"limit": {fmt.Sprint(*limit)}}
		for key, value := range map[string]string{"mode": *mode, "status": *status, "solver": *solver, "from": *from, "to": *to} {
			if strings.TrimSpace(value) != "" {
				values.Set(key, value)
			}
		}
		if strings.TrimSpace(*pageToken) != "" {
			values.Set("pageToken", *pageToken)
		}
		response, err = apiClient.Get(context.Background(), "/api/laps/optimizer-runs", values)
	case "get":
		if strings.TrimSpace(*runID) == "" {
			return writeConfigError(stdout, "--run-id is required")
		}
		response, err = apiClient.Get(context.Background(), "/api/laps/optimizer-runs/"+url.PathEscape(*runID), nil)
	default:
		return writeConfigError(stdout, "unknown scheduling-policy runs action: "+args[0])
	}
	if err != nil {
		return writeAPIError(stdout, err, command, nil, nil)
	}
	writeSuccess(stdout, response, map[string]any{"command": command})
	return ExitOK
}

func writeSchedulingPolicyUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  laps-cli scheduling-policy list [--status draft|published|archived]")
	fmt.Fprintln(w, "  laps-cli scheduling-policy get --id ID")
	fmt.Fprintln(w, "  laps-cli scheduling-policy create --file JSON")
	fmt.Fprintln(w, "  laps-cli scheduling-policy update --id ID --file JSON")
	fmt.Fprintln(w, "  laps-cli scheduling-policy delete --id ID --yes")
	fmt.Fprintln(w, "  laps-cli scheduling-policy clone|validate|publish --id ID")
	fmt.Fprintln(w, "  laps-cli scheduling-policy runs list|get")
}
