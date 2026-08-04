---
name: laps-workbuddy-mcp
description: Configure or diagnose the LAPS WorkBuddy MCP connector. Use when a user needs to install LAPS for WorkBuddy, set the APS system address, connect the same local account, trust the connector, update it, or resolve a connector readiness issue.
---

# LAPS WorkBuddy Connector

Use this skill only for installing, connecting, updating, or diagnosing the WorkBuddy connector. Use the relevant business skill for orders, materials, readiness, scheduling, capacity, or master data.

1. Ask for the organisation's APS system address if it is not already configured. Do not guess an address.
2. Install with the public package, then configure the address and have the user complete account login in the same operating-system account.
3. Generate the WorkBuddy configuration with `laps-mcp workbuddy config --print`; use `--install --yes` only after the user agrees to write the configuration.
4. Remind the user to open WorkBuddy's custom connector page and Trust the LAPS connector.
5. For a scheduling query or trial schedule, hand off to `$production-scheduling`. It must use the standard chart returned by the scheduling preview. Never prepare a local HTML page from order data as a substitute; only the connector's official Gantt result may be presented as a scheduling chart.
6. Use `laps-mcp status` to diagnose readiness. Explain failures as missing address, incomplete login, missing permission, or unavailable APS service. Do not show command output, protocol details, credentials, or raw errors unless technical diagnosis is explicitly requested.
