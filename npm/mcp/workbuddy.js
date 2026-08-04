import { chmod, copyFile, mkdir, readFile, stat, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { configuredServerURL, defaultBinDir, installationSettings, resolveTarget } from "../lib/launcher.js";

function workBuddyConfigPath(platform = process.platform, home = os.homedir()) {
  return path.join(home, ".workbuddy", "mcp.json");
}

async function mcpCommandPath() {
  const settings = await installationSettings();
  const target = resolveTarget();
  return path.join(settings.binDir || defaultBinDir(settings.installDir, target), target.goos === "windows" ? "laps-mcp.cmd" : "laps-mcp");
}

export async function buildWorkBuddyConfig() {
  const command = await mcpCommandPath();
  const server = await configuredServerURL();
  return {
    mcpServers: {
      laps: {
        command,
        args: [],
        env: server ? { SCHEDULING_API_BASE_URL: server } : {},
      },
    },
  };
}

async function readConfig(configPath) {
  try { return JSON.parse(await readFile(configPath, "utf8")); } catch (error) {
    if (error.code === "ENOENT") return {};
    throw new Error("WorkBuddy 的连接器配置无法读取，请先修复配置文件后重试。");
  }
}

export async function installWorkBuddyConfig({ yes = false, configPath = workBuddyConfigPath() } = {}) {
  if (!yes) throw new Error("写入 WorkBuddy 连接器前需要明确确认。");
  const current = await readConfig(configPath);
  const snippet = await buildWorkBuddyConfig();
  await mkdir(path.dirname(configPath), { recursive: true, mode: 0o700 });
  try {
    await stat(configPath);
    await copyFile(configPath, `${configPath}.backup-${Date.now()}`);
  } catch (error) { if (error.code !== "ENOENT") throw error; }
  const merged = { ...current, mcpServers: { ...(current.mcpServers || {}), ...snippet.mcpServers } };
  await writeFile(configPath, `${JSON.stringify(merged, null, 2)}\n`, { mode: 0o600 });
  await chmod(configPath, 0o600).catch(() => {});
  return configPath;
}

export async function runWorkBuddyConfig(args) {
  const install = args.includes("--install");
  const yes = args.includes("--yes");
  if (!args.includes("config") || args.some((arg) => !["config", "--print", "--install", "--yes"].includes(arg))) {
    throw new Error("用法：laps-mcp workbuddy config --print，或 laps-mcp workbuddy config --install --yes");
  }
  if (install) {
    const target = await installWorkBuddyConfig({ yes });
    process.stdout.write(`WorkBuddy 连接器已写入 ${target}。请在 WorkBuddy 中信任该连接器。\n`);
    return;
  }
  process.stdout.write(`${JSON.stringify(await buildWorkBuddyConfig(), null, 2)}\n`);
}
