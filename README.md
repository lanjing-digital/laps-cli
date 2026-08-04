# LAPS CLI

LAPS CLI is the command-line client for the LAPS APS services. It calls the authenticated `/api/laps/*` boundary; scheduling, capacity and readiness rules continue to run on the server.

The package includes seven focused business skills and a separate WorkBuddy connector skill:

- `laps-cli-auth`
- `laps-orders`
- `laps-material-master`
- `laps-material-readiness`
- `production-scheduling`
- `laps-capacity`
- `laps-master-data`
- `laps-workbuddy-mcp`

## Install with npx

Install the native CLI, WorkBuddy connector, and all skills for the current platform from the public GitHub repository. Provide the remote APS server address during installation:

```sh
npx --yes github:lanjing-digital/laps-cli install --server https://aps.example.com
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

`--base-url` is a one-command override, and `SCHEDULING_API_BASE_URL` takes priority over the saved setting. Use `laps-cli --help` and each command's `--help` for the current interface. Agent-specific installation guidance is in [docs/AGENT_INSTALL.md](docs/AGENT_INSTALL.md). For WorkBuddy, see [docs/WORKBUDDY_MCP.md](docs/WORKBUDDY_MCP.md).

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
```

`github` updates from this public repository. `npm` uses the public `@lanjing-digital/laps-cli` package once it is published. The default `auto` mode tries npm first and then falls back to GitHub.

## Release process

Push a version tag such as `v0.1.0`. GitHub Actions runs Go and launcher tests, cross-compiles the six supported platform targets, and publishes checksum-protected release assets. After the release completes, publish the npm package:

```sh
npm publish --access public
```

The package version and Git tag must match. Configure npm publishing credentials separately; this repository deliberately does not store registry tokens.
