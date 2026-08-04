#!/usr/bin/env node

import { connectionStatus } from "../mcp/business.js";
import { serve } from "../mcp/server.js";
import { runWorkBuddyConfig } from "../mcp/workbuddy.js";

const args = process.argv.slice(2);

try {
  if (args[0] === "workbuddy") {
    await runWorkBuddyConfig(args.slice(1));
  } else if (args[0] === "status") {
    const result = await connectionStatus();
    process.stdout.write(`${result.message}${result.guidance ? ` ${result.guidance}` : ""}\n`);
    process.exitCode = result.success ? 0 : 2;
  } else if (args.length === 0 || args[0] === "serve") {
    await serve("0.1.9");
  } else if (args.includes("--help") || args.includes("-h")) {
    process.stdout.write("Usage:\n  laps-mcp                         Start the WorkBuddy MCP server\n  laps-mcp status                  Check APS setup and login\n  laps-mcp workbuddy config --print\n  laps-mcp workbuddy config --install --yes\n");
  } else {
    throw new Error("未识别的连接器操作。可使用 laps-mcp --help 查看可用功能。");
  }
} catch (error) {
  process.stderr.write(`laps-mcp: ${error.message}\n`);
  process.exitCode = 1;
}
