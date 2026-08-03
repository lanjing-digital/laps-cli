# LAPS CLI

LAPS CLI is the command-line client for the LAPS APS services. It calls the authenticated `/api/laps/*` boundary; scheduling, capacity and readiness rules continue to run on the server.

The package includes seven focused agent skills:

- `laps-cli-auth`
- `laps-orders`
- `laps-material-master`
- `laps-material-readiness`
- `production-scheduling`
- `laps-capacity`
- `laps-master-data`

## Install with npx

Install the native CLI for the current platform and all skills:

```sh
npx @lanjing-digital/laps-cli@latest install
```

The command installs the executable to `~/.local/bin` on macOS/Linux and to the per-user local application directory on Windows. It installs skills to `~/.agents/skills` by default. Neither location is added to `PATH` automatically.

Choose locations explicitly when needed:

```sh
npx @lanjing-digital/laps-cli@latest install --bin-dir "$HOME/.local/bin" --skills-dir "$HOME/.codex/skills"
```

Install only selected skills, or only the CLI:

```sh
npx @lanjing-digital/laps-cli@latest install laps-orders laps-capacity
npx @lanjing-digital/laps-cli@latest install --no-skills
```

The launcher supports macOS, Linux and Windows on x64 and arm64. It downloads the matching Go binary from the public GitHub Release and verifies its SHA-256 checksum before running it.

## Run without a persistent installation

```sh
npx @lanjing-digital/laps-cli@latest auth status
npx @lanjing-digital/laps-cli@latest orders list
```

Set `SCHEDULING_API_BASE_URL` to the LAPS service URL, then authenticate with `laps-cli auth login`. Use `laps-cli --help` and each command's `--help` for the current interface.

## Release process

Push a version tag such as `v0.1.0`. GitHub Actions runs Go and launcher tests, cross-compiles the six supported platform targets, and publishes checksum-protected release assets. After the release completes, publish the npm package:

```sh
npm publish --access public
```

The package version and Git tag must match. Configure npm publishing credentials separately; this repository deliberately does not store registry tokens.
