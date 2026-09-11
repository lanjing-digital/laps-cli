# LAPS CLI

LAPS CLI is the command-line client for the LAPS APS services. It calls the authenticated `/api/laps/*` boundary; scheduling, capacity and readiness rules continue to run on the server.

The package includes eight focused business skills and a separate WorkBuddy connector skill:

- `laps-cli-auth`
- `laps-orders`
- `laps-material-master`
- `laps-material-readiness`
- `production-scheduling`
- `laps-capacity`
- `laps-master-data`
- `laps-scheduling-policy`
- `laps-workbuddy-mcp`

## Install with npx

Install the native CLI, WorkBuddy connector, and all skills for the current platform from the public GitHub repository. Provide the remote APS server address during installation:

```sh
npx --yes @lanjing-digital/laps-cli@latest install --non-interactive --server https://aps.example.com
```

Use an IP address when appropriate, for example `http://192.168.1.20:3000`. The command installs a self-updating launcher to `~/.local/bin` on macOS/Linux and to the per-user local application directory on Windows. It installs skills to `~/.agents/skills` by default. Neither location is added to `PATH` automatically.

Choose locations explicitly when needed:

```sh
npx --yes github:lanjing-digital/laps-cli install --server https://aps.example.com --bin-dir "$HOME/.local/bin" --skills-dir "$HOME/.codex/skills"
```

Install only selected skills, or only the CLI:

```sh
npx --yes github:lanjing-digital/laps-cli install laps-orders laps-capacity --server https://aps.example.com
npx --yes github:lanjing-digital/laps-cli install --no-skills --server https://aps.example.com
```

The launcher supports macOS, Linux and Windows on x64 and arm64. It downloads the matching precompiled binary from the public GitHub Release and verifies its SHA-256 checksum before running it. Users need Node.js 18 or newer for `npx`, but **do not need Go, a Go compiler, or a Go environment variable**.

## Run without a persistent installation

```sh
npx --yes github:lanjing-digital/laps-cli auth status --base-url https://aps.example.com
npx --yes github:lanjing-digital/laps-cli orders list --base-url https://aps.example.com
```

The first persistent use requires a remote APS server. Configure it once, then authenticate:

```sh
laps-cli config set-server --url https://aps.example.com
laps-cli auth login
```

`--base-url` is a one-command override, and `SCHEDULING_API_BASE_URL` takes priority over the saved setting. Agent-specific installation guidance is in [docs/AGENT_INSTALL.md](docs/AGENT_INSTALL.md). For WorkBuddy, see [docs/WORKBUDDY_MCP.md](docs/WORKBUDDY_MCP.md).

## WorkBuddy MCP connector

The installer also creates `laps-mcp`, a standard MCP stdio connector. It uses the same remote APS address and the same operating-system account's LAPS login as `laps-cli`; it does not require Go or a second token.

After installing and completing `laps-cli auth login`, print the WorkBuddy configuration:

```sh
laps-mcp workbuddy config --print
```

Or write it with an explicit confirmation:

```sh
laps-mcp workbuddy config --install --yes
```

Open WorkBuddy's custom connector settings afterwards and Trust the LAPS connector. The connector reports business outcomes and hides technical diagnostics by default.

For schedule queries and trial schedules, the connector uses the local LAPS installation to create an HTML Gantt chart and includes it as an attachment in the WorkBuddy conversation. Leave the display format and local output location unspecified to use this default.

## Update

```sh
laps-cli update --source github
laps-cli update --source npm
laps-cli update
laps-cli update --check --json
laps-cli version --json
```

`auto` checks npm first and uses GitHub if npm version discovery fails. The selected exact release is installed; `--force` reinstalls even at the current version. `--check` only queries versions and local skill markers, without installing or writing configuration. Updates preserve server settings, credentials, installation locations and selected skills (including `--no-skills`). An npm global/WorkBuddy-managed installation stays under the package manager's prefix.

Like [larksuite/cli](https://github.com/larksuite/cli/tree/main/internal/update), ordinary commands consult a 24-hour version cache and refresh it in the background. Warnings appear on stderr and under `_notice.update` / `_notice.skills` in JSON responses. Artifact bytes are unchanged. Skill installs record their release version; legacy, unknown or mismatched versions prompt synchronization. CI skips automatic notices; `LAPS_CLI_NO_UPDATE_NOTIFIER` and `LAPS_CLI_NO_SKILLS_NOTIFIER` opt out separately. Explicit `update --check` remains available.

## WorkBuddy CLI + Skill distribution

This is a separate connector option from the MCP configuration above. Generate a directory with `connector-meta.json`, `cli.json`, icon and eight source-derived business skills:

```sh
node scripts/build-workbuddy-connector.mjs ./dist/laps-workbuddy-cli
```

Optional third argument: an HTTPS server origin. The default is `https://lanjingshuzi.cn:3000` (general demo). Existing configured addresses are preserved. End users can change it with `laps-cli config set-server --url URL` and reconnect.

WorkBuddy supplies Node.js 20. Its init command installs the same npm package and uses `install --managed --non-interactive --no-skills` to prepare the checksum-verified binary in the managed package directory. Credentials remain outside that directory. `auth`, `status` and `unAuth` config entries invoke existing `auth login --no-browser`, `auth status --local` and `auth logout`; no top-level aliases are added. `authWaitForExit: true` preserves the OAuth callback process. Local status reports a persisted renewable session only; it does not prove remote authorization has not been revoked.

## Release process

Canonical sources live in the main APS repository's `cli/`, including the npm launcher and release workflow. Synchronize that directory's declared distribution files into the public repository. `VERSION`, package.json, package-lock.json and the release tag must match. GitHub Actions tests, builds six binaries, publishes the checksum-protected Release and then publishes npm through the existing CI secret. Rebuild and resubmit the WorkBuddy connector for each release so its skills and install pin remain aligned.
