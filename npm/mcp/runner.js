import { spawn } from "node:child_process";
import { ensureBinary, configuredServerURL } from "../lib/launcher.js";

export class LapsCommandError extends Error {
  constructor(message, { stdout = "", stderr = "", status = 1 } = {}) {
    super(message);
    this.stdout = stdout;
    this.stderr = stderr;
    this.status = status;
  }
}

function parseJSON(stdout) {
  try { return JSON.parse(stdout.trim()); } catch { return undefined; }
}

function redact(value) {
  return String(value || "")
    .replace(/https?:\/\/[^\s"']+/gi, "[远端地址]")
    .replace(/(?:token|authorization|bearer)[=: ]+[^\s"']+/gi, "$1=[已隐藏]");
}

export async function runLapsCommand(args, environment = process.env) {
  const baseURL = String(environment.SCHEDULING_API_BASE_URL || "").trim() || await configuredServerURL();
  if (!baseURL) throw new LapsCommandError("APS 系统地址尚未设置", { status: 2 });
  const executable = environment.LAPS_MCP_CLI_PATH || await ensureBinary();
  const finalArgs = [...args, "--base-url", baseURL];
  const child = spawn(executable, finalArgs, { shell: false, windowsHide: true, env: environment, stdio: ["ignore", "pipe", "pipe"] });
  let stdout = "";
  let stderr = "";
  child.stdout.setEncoding("utf8");
  child.stderr.setEncoding("utf8");
  child.stdout.on("data", (chunk) => { stdout += chunk; });
  child.stderr.on("data", (chunk) => { stderr += chunk; });
  const status = await new Promise((resolve, reject) => {
    child.once("error", reject);
    child.once("close", resolve);
  });
  const data = parseJSON(stdout);
  if (status !== 0 || data?.success === false) {
    process.stderr.write(`[laps-mcp] LAPS request failed: ${redact(stderr || stdout)}\n`);
    throw new LapsCommandError("LAPS 业务操作未完成", { stdout, stderr, status });
  }
  return { data: data ?? { output: stdout.trim() }, output: stdout, diagnostics: redact(stderr), status };
}
