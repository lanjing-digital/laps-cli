import assert from "node:assert/strict";
import { chmod, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { spawn } from "node:child_process";
import { executeDomain } from "../mcp/business.js";
import { prepareInvocation } from "../mcp/operations.js";
import { installWorkBuddyConfig } from "../mcp/workbuddy.js";

test("MCP operation mapping only accepts named business operations and settings", async () => {
  const invocation = await prepareInvocation("orders", { operation: "list", options: { status: "pending", limit: 20 } });
  assert.deepEqual(invocation.args, ["orders", "list", "--status", "pending", "--limit", "20"]);
  await invocation.cleanup();
  await assert.rejects(() => prepareInvocation("orders", { operation: "list", options: { "base-url": "https://unexpected.example" } }), /不支持的设置项/);
  await assert.rejects(() => prepareInvocation("orders", { operation: "import-apply", payload: {}, confirm: false }), /取得确认/);
  await assert.rejects(() => prepareInvocation("orders", { operation: "delete", id: "order_1", confirm: true }), /删除前需要再次确认/);
});

test("MCP creates and removes a private JSON payload file", async () => {
  const invocation = await prepareInvocation("orders", { operation: "create", payload: { sequenceNo: "FG-001" }, confirm: true });
  const fileIndex = invocation.args.indexOf("--file") + 1;
  const payloadPath = invocation.args[fileIndex];
  assert.deepEqual(JSON.parse(await readFile(payloadPath, "utf8")), { sequenceNo: "FG-001" });
  await invocation.cleanup();
  await assert.rejects(() => readFile(payloadPath, "utf8"));
});

test("scheduling previews create a private HTML chart by default", async () => {
  const invocation = await prepareInvocation("scheduling", { operation: "auto-preview" });
  assert.equal(invocation.args.at(-4), "--format");
  assert.equal(invocation.args.at(-3), "html");
  assert.equal(invocation.args.at(-2), "--output");
  assert.match(invocation.args.at(-1), /schedule-preview\.html$/);
  assert.equal(invocation.previewArtifact?.mimeType, "text/html");
  await invocation.cleanup();

  const explicitHtml = await prepareInvocation("scheduling", { operation: "auto-preview", format: "html" });
  assert.equal(explicitHtml.args.filter((item) => item === "--format").length, 1);
  assert.equal(explicitHtml.previewArtifact?.mimeType, "text/html");
  await explicitHtml.cleanup();
});

test("scheduling preview exposes solver choices and safe preview application", async () => {
  const preview = await prepareInvocation("scheduling", { operation: "auto-preview", options: { "solver-mode": "portfolio", "include-candidate-plans": false } });
  assert.ok(preview.args.includes("--solver-mode"));
  assert.ok(preview.args.includes("portfolio"));
  assert.ok(preview.args.includes("--include-candidate-plans"));
  assert.ok(preview.args.includes("false"));
  await preview.cleanup();

  const apply = await prepareInvocation("scheduling", { operation: "auto-apply-preview", previewToken: "preview-token", candidateSolver: "cp-sat", confirm: true });
  assert.deepEqual(apply.args, ["auto-schedule", "apply", "--preview-token", "preview-token", "--candidate-solver", "cp-sat"]);
  await apply.cleanup();
  await assert.rejects(() => prepareInvocation("scheduling", { operation: "auto-apply-preview", previewToken: "preview-token", candidateSolver: "cp-sat" }), /取得确认/);
});

test("business result hides raw command output", { concurrency: false }, async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "laps-mcp-test-"));
  const fakeCLI = path.join(directory, "laps-cli");
  await writeFile(fakeCLI, "#!/usr/bin/env node\nconsole.log(JSON.stringify({success:true, records:[{id:'order_1'}], total:1, token:'not-for-user'}));\n", { mode: 0o700 });
  await chmod(fakeCLI, 0o700);
  const originalPath = process.env.LAPS_MCP_CLI_PATH;
  const originalURL = process.env.SCHEDULING_API_BASE_URL;
  process.env.LAPS_MCP_CLI_PATH = fakeCLI;
  process.env.SCHEDULING_API_BASE_URL = "https://aps.example.com";
  try {
    const result = await executeDomain("orders", { operation: "list" });
    assert.equal(result.success, true);
    assert.match(result.message, /生产订单已查询完成/);
    assert.doesNotMatch(result.message, /token|laps-cli|https/i);
  } finally {
    if (originalPath === undefined) delete process.env.LAPS_MCP_CLI_PATH; else process.env.LAPS_MCP_CLI_PATH = originalPath;
    if (originalURL === undefined) delete process.env.SCHEDULING_API_BASE_URL; else process.env.SCHEDULING_API_BASE_URL = originalURL;
    await rm(directory, { recursive: true, force: true });
  }
});

test("scheduling preview returns the locally generated HTML as an MCP resource payload", { concurrency: false }, async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "laps-mcp-chart-test-"));
  const fakeCLI = path.join(directory, "laps-cli");
  await writeFile(fakeCLI, "#!/usr/bin/env node\nconst output = process.argv[process.argv.indexOf('--output') + 1];\nawait (await import('node:fs/promises')).writeFile(output, '<!doctype html><title>排产图</title>');\nconsole.log(JSON.stringify({success:true,total:2}));\n", { mode: 0o700 });
  await chmod(fakeCLI, 0o700);
  const originalPath = process.env.LAPS_MCP_CLI_PATH;
  const originalURL = process.env.SCHEDULING_API_BASE_URL;
  process.env.LAPS_MCP_CLI_PATH = fakeCLI;
  process.env.SCHEDULING_API_BASE_URL = "https://aps.example.com";
  try {
    const result = await executeDomain("scheduling", { operation: "auto-preview" });
    assert.equal(result.success, true);
    assert.equal(result.previewArtifact?.mimeType, "text/html");
    assert.match(result.previewArtifact?.html || "", /排产图/);
    assert.match(result.message, /排产图已附在本次对话中/);
  } finally {
    if (originalPath === undefined) delete process.env.LAPS_MCP_CLI_PATH; else process.env.LAPS_MCP_CLI_PATH = originalPath;
    if (originalURL === undefined) delete process.env.SCHEDULING_API_BASE_URL; else process.env.SCHEDULING_API_BASE_URL = originalURL;
    await rm(directory, { recursive: true, force: true });
  }
});

test("MCP stdio scheduling preview embeds the HTML chart as a resource", { concurrency: false }, async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "laps-mcp-stdio-chart-test-"));
  const fakeCLI = path.join(directory, "laps-cli");
  await writeFile(fakeCLI, "#!/usr/bin/env node\nconst output = process.argv[process.argv.indexOf('--output') + 1];\nawait (await import('node:fs/promises')).writeFile(output, '<!doctype html><title>排产图</title>');\nconsole.log(JSON.stringify({success:true,total:2}));\n", { mode: 0o700 });
  await chmod(fakeCLI, 0o700);
  const child = spawn(process.execPath, ["npm/bin/laps-mcp.js"], {
    cwd: process.cwd(),
    env: { ...process.env, LAPS_MCP_CLI_PATH: fakeCLI, SCHEDULING_API_BASE_URL: "https://aps.example.com" },
    stdio: ["pipe", "pipe", "pipe"],
  });
  let output = "";
  child.stdout.setEncoding("utf8");
  child.stdout.on("data", (chunk) => { output += chunk; });
  try {
    child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2025-03-26", capabilities: {}, clientInfo: { name: "test", version: "1" } } })}\n`);
    for (let attempt = 0; attempt < 20 && !output.includes('"id":1'); attempt += 1) await new Promise((resolve) => setTimeout(resolve, 50));
    child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized", params: {} })}\n`);
    await new Promise((resolve) => setTimeout(resolve, 50));
    child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/call", params: { name: "laps_scheduling", arguments: { operation: "auto-preview" } } })}\n`);
    for (let attempt = 0; attempt < 20 && !output.includes('"id":2'); attempt += 1) await new Promise((resolve) => setTimeout(resolve, 50));
    const message = output.trim().split(/\n/).map((line) => JSON.parse(line)).find((item) => item.id === 2);
    const resource = message?.result?.content?.find((item) => item.type === "resource")?.resource;
    assert.equal(resource?.mimeType, "text/html");
    assert.match(resource?.text || "", /排产图/);
  } finally {
    child.kill();
    await rm(directory, { recursive: true, force: true });
  }
});

test("WorkBuddy configuration requires explicit write confirmation and preserves existing servers", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "laps-workbuddy-test-"));
  const configPath = path.join(directory, "mcp.json");
  await writeFile(configPath, JSON.stringify({ mcpServers: { existing: { command: "existing" } } }));
  await assert.rejects(() => installWorkBuddyConfig({ configPath }), /明确确认/);
  const originalURL = process.env.SCHEDULING_API_BASE_URL;
  process.env.SCHEDULING_API_BASE_URL = "https://aps.example.com";
  try {
    await installWorkBuddyConfig({ configPath, yes: true });
    const configured = JSON.parse(await readFile(configPath, "utf8"));
    assert.equal(configured.mcpServers.existing.command, "existing");
    assert.match(configured.mcpServers.laps.command, /laps-mcp/);
    assert.equal(configured.mcpServers.laps.env.SCHEDULING_API_BASE_URL, "https://aps.example.com");
  } finally {
    if (originalURL === undefined) delete process.env.SCHEDULING_API_BASE_URL; else process.env.SCHEDULING_API_BASE_URL = originalURL;
    await rm(directory, { recursive: true, force: true });
  }
});

test("MCP stdio server completes initialize and exposes all fixed tools", async () => {
  const child = spawn(process.execPath, ["npm/bin/laps-mcp.js"], { cwd: process.cwd(), stdio: ["pipe", "pipe", "pipe"] });
  let output = "";
  child.stdout.setEncoding("utf8");
  child.stdout.on("data", (chunk) => { output += chunk; });
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2025-03-26", capabilities: {}, clientInfo: { name: "test", version: "1" } } })}\n`);
  for (let attempt = 0; attempt < 20 && !output.includes('"id":1'); attempt += 1) await new Promise((resolve) => setTimeout(resolve, 50));
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized", params: {} })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} })}\n`);
  for (let attempt = 0; attempt < 20 && !output.includes('"id":2'); attempt += 1) await new Promise((resolve) => setTimeout(resolve, 50));
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 3, method: "tools/call", params: { name: "laps_orders", arguments: { operation: "import-apply", payload: {} } } })}\n`);
  for (let attempt = 0; attempt < 20 && !output.includes('"id":3'); attempt += 1) await new Promise((resolve) => setTimeout(resolve, 50));
  child.kill();
  const messages = output.trim().split(/\n/).filter(Boolean).map((line) => JSON.parse(line));
  const tools = messages.find((message) => message.id === 2)?.result?.tools || [];
  assert.deepEqual(tools.map((tool) => tool.name), ["laps_connection", "laps_orders", "laps_material_master", "laps_material_readiness", "laps_scheduling", "laps_capacity", "laps_master_data", "laps_scheduling_policy"]);
  const mutation = messages.find((message) => message.id === 3)?.result;
  assert.equal(mutation.isError, true);
  assert.match(mutation.content[0].text, /取得确认/);
});
