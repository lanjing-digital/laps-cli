---
name: laps-cli-auth
description: Authenticate laps-cli with the current user's browser session. Use for LAPS CLI login, OAuth status, logout, expired credentials, missing scopes, or switching the personally authorized account.
---

# LAPS CLI Authentication

Use `laps-cli auth login|status|logout`. Login reuses the browser session and stores only the current user's refreshable OAuth credential.

## Workflow

1. Run `laps-cli auth status` before a protected business command when authentication is uncertain.
2. Run `laps-cli auth login` when credentials are missing, expired, or do not contain newly required scopes.
3. Run `laps-cli auth logout` only when the user asks to revoke and remove the local credential.
4. Use `laps-cli auth ... --help` for exact flags.

Never copy another user's credential or substitute a shared system token. Remote base URLs must use HTTPS; loopback HTTP is only for local development.
