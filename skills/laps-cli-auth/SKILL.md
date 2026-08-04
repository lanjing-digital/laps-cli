---
name: laps-cli-auth
description: Authenticate laps-cli with the current user's browser session. Use for LAPS CLI login, OAuth status, logout, expired credentials, missing scopes, or switching the personally authorized account.
---

# LAPS CLI Authentication

Use `laps-cli auth login|status|logout`. Login reuses the browser session and stores only the current user's refreshable OAuth credential.

When running in WorkBuddy, use `laps_connection` to check readiness. Do not attempt login inside the connector: guide the user to complete it in the same computer account first.

## Workflow

1. Run `laps-cli auth status` before a protected business command when authentication is uncertain.
2. Run `laps-cli auth login` when credentials are missing, expired, or do not contain newly required scopes.
3. Run `laps-cli auth logout` only when the user asks to revoke and remove the local credential.
4. Use `laps-cli auth ... --help` for exact flags.

Never copy another user's credential or substitute a shared system token. Remote base URLs must use HTTPS; loopback HTTP is only for local development.

## Business communication

- Tell the user “账号已准备就绪” or “请先完成账号登录”，not token, OAuth, command, or raw error details.
- Ask for the APS system address in ordinary business language when it has not been set; never guess it.
- Mention technical diagnostics only when the user explicitly requests them, and never expose credentials.
