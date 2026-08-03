# LAPS CLI installation for agents

Use this document when an agent needs to install or initialize LAPS CLI for a user. The CLI invokes the user's permitted `/api/laps/*` operations; it does not contain scheduling rules.

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

It installs the self-updating launcher and all seven domain skills. By default, skills go to `~/.agents/skills`. To install for Codex instead:

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
