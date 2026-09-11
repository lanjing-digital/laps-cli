package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	buildinfo "production-scheduling-cli"
	cliAuth "production-scheduling-cli/internal/auth"
	"production-scheduling-cli/internal/maintenance"
)

type noticeWriter struct {
	io.Writer
	checker maintenance.Checker
}

func newChecker() maintenance.Checker {
	credentials, _ := cliAuth.DefaultCredentialsPath()
	home, _ := os.UserHomeDir()
	skillsDir := os.Getenv("LAPS_SKILLS_DIR")
	if skillsDir == "" {
		skillsDir = filepath.Join(home, ".agents", "skills")
	}
	return maintenance.Checker{
		Version: buildinfo.Version(), ConfigDir: filepath.Dir(credentials), SkillsDir: skillsDir,
		BundleVersion: os.Getenv("LAPS_CLI_SKILLS_VERSION"),
		SkillNames:    []string{"laps-cli-auth", "laps-orders", "laps-material-master", "laps-material-readiness", "production-scheduling", "laps-capacity", "laps-master-data", "laps-scheduling-policy", "laps-workbuddy-mcp"},
	}
}

// RunWithNotices keeps advice separate from command success and artifact bytes.
func RunWithNotices(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || wantsHelp(args) || args[0] == "__refresh-update-cache" || args[0] == "update" || args[0] == "version" || args[0] == "--version" || args[0] == "-v" || (args[0] == "auth" && len(args) > 1 && args[1] == "status") {
		return Run(args, stdout, stderr)
	}
	for _, key := range []string{"CI", "BUILD_NUMBER", "RUN_ID"} {
		if os.Getenv(key) != "" {
			return Run(args, stdout, stderr)
		}
	}
	checker := newChecker()
	if os.Getenv("LAPS_CLI_NO_UPDATE_NOTIFIER") == "" && os.Getenv("LAPS_CLI_CACHE_REFRESH_STARTED") == "" {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go checker.Refresh(ctx)
	}
	wrapped := noticeWriter{Writer: stdout, checker: checker}
	code := Run(args, wrapped, stderr)
	for key, notice := range wrapped.notices() {
		fmt.Fprintf(stderr, "[%s] %s\n", key, notice.Message)
	}
	return code
}

func (w noticeWriter) notices() map[string]maintenance.Notice {
	notices := w.checker.CachedNotices()
	if os.Getenv("LAPS_CLI_NO_UPDATE_NOTIFIER") != "" {
		delete(notices, "update")
	}
	if os.Getenv("LAPS_CLI_NO_SKILLS_NOTIFIER") != "" {
		delete(notices, "skills")
	}
	return notices
}

func addNotices(w io.Writer, payload map[string]any) {
	if writer, ok := w.(noticeWriter); ok {
		if notices := writer.notices(); len(notices) > 0 {
			payload["_notice"] = notices
		}
	}
}

func runVersion(args []string, stdout io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, buildinfo.Version())
		return ExitOK
	}
	if len(args) == 1 && args[0] == "--json" {
		writeJSONMap(stdout, map[string]any{"version": buildinfo.Version()})
		return ExitOK
	}
	if wantsHelp(args) {
		fmt.Fprintln(stdout, "Usage: laps-cli version [--json]")
		return ExitOK
	}
	return writeConfigError(stdout, "Usage: laps-cli version [--json]")
}

func runUpdateCheck(args []string, stdout io.Writer) int {
	if wantsHelp(args) {
		fmt.Fprintln(stdout, "Usage: laps-cli update [--check] [--json] [--force] [--source auto|npm|github]\nInstall/update execution is provided by the npm launcher; standalone binaries support --check.")
		return ExitOK
	}
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	check := fs.Bool("check", false, "check without installing")
	_ = fs.Bool("json", false, "structured output")
	source := fs.String("source", "auto", "version source")
	if code := parseNoArgs(fs, args, stdout, "update"); code != ExitOK {
		return code
	}
	if !*check {
		return writeConfigError(stdout, "For installation, use: npx --yes @lanjing-digital/laps-cli@latest install --non-interactive; this standalone binary supports update --check --json")
	}
	result, err := newChecker().Check(context.Background(), *source)
	if err != nil {
		return writeConfigError(stdout, err.Error())
	}
	_ = json.NewEncoder(stdout).Encode(result)
	return ExitOK
}

func sameServer(a, b string) bool { return strings.TrimRight(a, "/") == strings.TrimRight(b, "/") }
