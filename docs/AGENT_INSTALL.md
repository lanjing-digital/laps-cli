# LAPS CLI installation for agents

Use this document when an agent needs to install or initialize LAPS CLI for a user. The CLI invokes the user's permitted `/api/laps/*` operations; it does not contain scheduling rules.

## Prerequisite

The user needs Node.js 18 or newer because installation runs through `npx`. They do **not** need Go, a Go compiler, or any Go environment configuration. The installer downloads the matching checksum-verified native binary from the public GitHub Release.

## Required information

Before installation, obtain the APS server address from the user or their administrator. It must be a complete HTTP(S) URL, including the scheme and optional port.

- Domain example: `https://aps.example.com`
- Private-network IP example: `http://192.168.1.20:3000`

Do not guess a production address or silently use `127.0.0.1`. If the address is unknown, ask the user. A one-command address can be supplied with `--base-url`; the persistent configuration is stored in the current user's standard application configuration directory.

## Install CLI and skills

The public GitHub form is available immediately and is the canonical bootstrap path:

```sh
npx --yes github:lanjing-digital/laps-cli install --server https://aps.example.com
```

It installs the self-updating `laps-cli` and `laps-mcp` launchers, seven domain skills, and the WorkBuddy connector skill. By default, skills go to `~/.agents/skills`. To install for Codex instead:

```sh
npx --yes github:lanjing-digital/laps-cli install --server https://aps.example.com --skills-dir "$HOME/.codex/skills"
```

To configure a server after installation or change it later:

```sh
laps-cli config set-server --url https://aps.example.com
laps-cli config show-server
```

## First use

Confirm the configured server and start the user's OAuth login:

```sh
laps-cli config show-server
laps-cli auth login
```

If the server has not been configured, the launcher stops before sending a request and explains how to set it. For a single command, do not persist a setting; pass `--base-url` explicitly:

```sh
laps-cli orders list --base-url https://aps.example.com
```

`SCHEDULING_API_BASE_URL` is also supported for managed environments and takes precedence over the saved address.

## WorkBuddy

WorkBuddy must run under the same operating-system account that installed LAPS. First configure the APS address and complete the user's login outside the connector:

```sh
laps-cli config show-server
laps-cli auth login
laps-mcp status
```

Print the configuration for `~/.workbuddy/mcp.json`:

```sh
laps-mcp workbuddy config --print
```

Or write it only after the user confirms; the command backs up an existing file before merging the `laps` server entry:

```sh
laps-mcp workbuddy config --install --yes
```

Restart or reload WorkBuddy, then open its custom connector management page and select **Trust** for LAPS. The connector never opens a browser login flow; when login expires, have the user run `laps-cli auth login` in their terminal again.

## Update

Use the GitHub source for a public-repository update:

```sh
laps-cli update --source github
```

The updater also supports the npm distribution once the scoped package is published:

```sh
laps-cli update --source npm
```

`laps-cli update` uses `auto`: it tries npm first, then GitHub. Updating replaces the launcher package, retrieves the matching platform binary, re-installs the bundled skills, and preserves the configured server address.

## Safety rules for agents

- Run `preview` before `apply` or `publish` when the target command supports both.
- Request the user's confirmation before any `apply`, `publish`, or delete operation.
- Do not put OAuth tokens in commands, files, or messages. Use `laps-cli auth login`.
- Keep skills separated by domain; install only the requested skill names when a full install is unnecessary.
- Respond in business language: state the result, affected scope, and next action. Do not expose command names, endpoints, tokens, or raw errors unless the user explicitly asks for technical diagnosis.
