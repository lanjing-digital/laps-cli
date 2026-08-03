import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { chmod, cp, mkdir, readFile, rename, rm, stat, writeFile } from "node:fs/promises";
import { accessSync, constants } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const require = createRequire(import.meta.url);
const packageInfo = require("../../package.json");
const packageRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const releaseRepository = process.env.LAPS_CLI_RELEASE_REPOSITORY || "lanjing-digital/laps-cli";
const installationFile = "installation.json";

export const allSkills = [
  "laps-cli-auth",
  "laps-orders",
  "laps-material-master",
  "laps-material-readiness",
  "production-scheduling",
  "laps-capacity",
  "laps-master-data",
];

export function resolveTarget(platform = process.platform, arch = process.arch) {
  const goos = { darwin: "darwin", linux: "linux", win32: "windows" }[platform];
  const goarch = { x64: "amd64", arm64: "arm64" }[arch];
  if (!goos || !goarch) {
    throw new Error(`unsupported platform: ${platform}/${arch}; supported: macOS, Linux, and Windows on x64 or arm64`);
  }
  const suffix = goos === "windows" ? ".exe" : "";
  return { goos, goarch, executable: `laps-cli${suffix}`, asset: `laps-cli_${goos}_${goarch}${suffix}` };
}

export function normalizeServerURL(value) {
  const raw = String(value || "").trim();
  if (!raw) throw new Error("server URL is required");
  let parsed;
  try {
    parsed = new URL(raw);
  } catch {
    throw new Error("server URL must include http:// or https://, for example https://aps.example.com");
  }
  if (!parsed.hostname || !["http:", "https:"].includes(parsed.protocol) || parsed.username || parsed.password || parsed.search || parsed.hash) {
    throw new Error("server URL must be an http(s) IP address or domain without credentials, query, or fragment");
  }
  return parsed.toString().replace(/\/$/, "");
}

export function defaultConfigDir(platform = process.platform, environment = process.env) {
  if (String(environment.LAPS_CLI_CONFIG_DIR || "").trim()) return environment.LAPS_CLI_CONFIG_DIR;
  const home = os.homedir();
  if (platform === "darwin") return path.join(home, "Library", "Application Support", "laps-cli");
  if (platform === "win32") return path.join(environment.APPDATA || path.join(home, "AppData", "Roaming"), "laps-cli");
  return path.join(environment.XDG_CONFIG_HOME || path.join(home, ".config"), "laps-cli");
}

function settingsPath() {
  return path.join(defaultConfigDir(), "settings.json");
}

async function readJSON(filePath, fallback = {}) {
  try {
    return JSON.parse(await readFile(filePath, "utf8"));
  } catch (error) {
    if (error.code === "ENOENT") return fallback;
    throw new Error(`read ${filePath}: ${error.message}`);
  }
}

async function saveSettings(baseURL) {
  const directory = defaultConfigDir();
  await mkdir(directory, { recursive: true, mode: 0o700 });
  await chmod(directory, 0o700).catch(() => {});
  const destination = settingsPath();
  const temporary = `${destination}.${process.pid}.tmp`;
  await writeFile(temporary, `${JSON.stringify({ baseUrl: normalizeServerURL(baseURL) }, null, 2)}\n`, { mode: 0o600 });
  await chmod(temporary, 0o600).catch(() => {});
  await rename(temporary, destination);
}

async function configuredServerURL() {
  const fromEnvironment = String(process.env.SCHEDULING_API_BASE_URL || "").trim();
  if (fromEnvironment) return normalizeServerURL(fromEnvironment);
  const settings = await readJSON(settingsPath());
  return settings.baseUrl ? normalizeServerURL(settings.baseUrl) : "";
}

function releaseURL(asset) {
  const tag = `v${packageInfo.version}`;
  return `https://github.com/${releaseRepository}/releases/download/${tag}/${asset}`;
}

function cachePath(target, root = packageRoot) {
  return path.join(root, "vendor", `${target.goos}-${target.goarch}`, target.executable);
}

async function download(url) {
  const response = await fetch(url);
  if (!response.ok) throw new Error(`download failed (${response.status}): ${url}`);
  return Buffer.from(await response.arrayBuffer());
}

function expectedChecksum(checksums, asset) {
  const match = checksums.split(/\r?\n/).find((line) => line.trim().endsWith(`  ${asset}`));
  if (!match) throw new Error(`checksum missing for ${asset}`);
  return match.trim().split(/\s+/)[0].toLowerCase();
}

export async function ensureBinary(root = packageRoot) {
  const target = resolveTarget();
  const destination = cachePath(target, root);
  try {
    accessSync(destination, constants.X_OK);
    return destination;
  } catch {
    // The package may be newly installed or its cache may have been cleared.
  }

  const releaseBase = releaseURL("").replace(/\/$/, "");
  const [checksums, binary] = await Promise.all([
    download(`${releaseBase}/checksums.txt`).then((value) => value.toString("utf8")),
    download(releaseURL(target.asset)),
  ]);
  const expected = expectedChecksum(checksums, target.asset);
  const actual = createHash("sha256").update(binary).digest("hex");
  if (actual !== expected) throw new Error(`checksum verification failed for ${target.asset}`);

  await mkdir(path.dirname(destination), { recursive: true });
  const temporary = `${destination}.${process.pid}.tmp`;
  await writeFile(temporary, binary, { mode: 0o755 });
  if (target.goos !== "windows") await chmod(temporary, 0o755);
  await rename(temporary, destination);
  return destination;
}

function takeOption(args, name, description = "a value") {
  const index = args.indexOf(name);
  if (index < 0) return undefined;
  if (!args[index + 1] || args[index + 1].startsWith("--")) throw new Error(`${name} requires ${description}`);
  const value = args[index + 1];
  args.splice(index, 2);
  return value;
}

function defaultInstallDir(target = resolveTarget()) {
  return target.goos === "windows"
    ? path.join(process.env.LOCALAPPDATA || os.homedir(), "laps-cli")
    : path.join(os.homedir(), ".local", "share", "laps-cli");
}

function defaultBinDir(installDir, target = resolveTarget()) {
  return target.goos === "windows" ? path.join(installDir, "bin") : path.join(os.homedir(), ".local", "bin");
}

function defaultSkillsDir() {
  return process.env.LAPS_SKILLS_DIR || path.join(os.homedir(), ".agents", "skills");
}

function validUpdateSource(source) {
  return ["auto", "npm", "github"].includes(source);
}

export function parseInstallArgs(argumentsList) {
  const args = [...argumentsList];
  const binDir = takeOption(args, "--bin-dir", "a directory");
  const skillsDir = takeOption(args, "--skills-dir", "a directory");
  const installDir = takeOption(args, "--install-dir", "a directory");
  const server = takeOption(args, "--server", "an http(s) URL");
  const source = takeOption(args, "--source", "auto, npm, or github") || "github";
  const noSkills = args.includes("--no-skills");
  if (noSkills) args.splice(args.indexOf("--no-skills"), 1);
  if (!validUpdateSource(source) || source === "auto") throw new Error("--source must be npm or github when installing");
  if (server) normalizeServerURL(server);
  if (args.some((value) => value.startsWith("--"))) throw new Error(`unknown install option: ${args.find((value) => value.startsWith("--"))}`);
  if (args.some((skill) => !allSkills.includes(skill))) throw new Error(`unknown skill: ${args.find((skill) => !allSkills.includes(skill))}`);
  if (noSkills && args.length > 0) throw new Error("skill names cannot be used with --no-skills");
  return { binDir, skillsDir, installDir, server, source, noSkills, skills: args.length ? args : allSkills };
}

export function parseUpdateArgs(argumentsList) {
  const args = [...argumentsList];
  const source = takeOption(args, "--source", "auto, npm, or github") || "auto";
  if (!validUpdateSource(source)) throw new Error("--source must be auto, npm, or github");
  if (args.length) throw new Error(`unknown update option: ${args[0]}`);
  return { source };
}

async function installSkill(skill, targetRoot, sourceRoot) {
  const source = path.join(sourceRoot, skill);
  await Promise.all([stat(path.join(source, "SKILL.md")), stat(path.join(source, "agents", "openai.yaml"))]);
  await mkdir(targetRoot, { recursive: true });
  const stagingRoot = path.join(targetRoot, `.laps-skill-install-${process.pid}-${Date.now()}`);
  const staging = path.join(stagingRoot, skill);
  const destination = path.join(targetRoot, skill);
  const backup = path.join(stagingRoot, "previous");
  try {
    await mkdir(stagingRoot, { recursive: true });
    await cp(source, staging, { recursive: true });
    await Promise.all([stat(path.join(staging, "SKILL.md")), stat(path.join(staging, "agents", "openai.yaml"))]);
    try { await stat(destination); await rename(destination, backup); } catch (error) { if (error.code !== "ENOENT") throw error; }
    await rename(staging, destination);
  } finally {
    await rm(stagingRoot, { recursive: true, force: true });
  }
  process.stdout.write(`installed ${skill} to ${destination}\n`);
}

function distributionFilter(source) {
  return ![".git", "node_modules", "vendor", "dist"].includes(path.basename(source));
}

async function installPackage(installDir) {
  if (path.resolve(installDir) === packageRoot) return packageRoot;
  const parent = path.dirname(installDir);
  const stagingRoot = path.join(parent, `.laps-cli-package-install-${process.pid}-${Date.now()}`);
  const staging = path.join(stagingRoot, "package");
  const backup = path.join(stagingRoot, "previous");
  try {
    await mkdir(stagingRoot, { recursive: true });
    await cp(packageRoot, staging, { recursive: true, filter: distributionFilter });
    await Promise.all([
      stat(path.join(staging, "npm", "bin", "laps-cli.js")),
      stat(path.join(staging, "npm", "lib", "launcher.js")),
      stat(path.join(staging, "skills", "laps-cli-auth", "SKILL.md")),
    ]);
    try { await stat(installDir); await rename(installDir, backup); } catch (error) { if (error.code !== "ENOENT") throw error; }
    await rename(staging, installDir);
  } finally {
    await rm(stagingRoot, { recursive: true, force: true });
  }
  return installDir;
}

function shellQuote(value) {
  return `'${String(value).replaceAll("'", `'"'"'`)}'`;
}

async function writeLauncher(installDir, binDir, target) {
  await mkdir(binDir, { recursive: true });
  const destination = path.join(binDir, target.goos === "windows" ? "laps-cli.cmd" : "laps-cli");
  const nodeEntrypoint = path.join(installDir, "npm", "bin", "laps-cli.js");
  const content = target.goos === "windows"
    ? `@echo off\r\nset "LAPS_CLI_INSTALL_DIR=${installDir.replaceAll('"', '""')}"\r\nnode "%LAPS_CLI_INSTALL_DIR%\\npm\\bin\\laps-cli.js" %*\r\n`
    : `#!/usr/bin/env sh\nexport LAPS_CLI_INSTALL_DIR=${shellQuote(installDir)}\nexec node ${shellQuote(nodeEntrypoint)} "$@"\n`;
  const temporary = `${destination}.${process.pid}.tmp`;
  await writeFile(temporary, content, { mode: 0o755 });
  if (target.goos !== "windows") await chmod(temporary, 0o755);
  await rename(temporary, destination);
  return destination;
}

async function configureServerDuringInstall(server) {
  if (server) {
    await saveSettings(server);
    process.stdout.write(`configured LAPS server: ${normalizeServerURL(server)}\n`);
    return;
  }
  const configured = await configuredServerURL();
  if (configured) {
    process.stdout.write(`using configured LAPS server: ${configured}\n`);
    return;
  }
  process.stdout.write("LAPS needs your remote APS server IP address or domain before first use.\n");
  process.stdout.write("Run `laps-cli config set-server --url https://aps.example.com` or set SCHEDULING_API_BASE_URL.\n");
}

async function install(argumentsList) {
  if (argumentsList.includes("--help") || argumentsList.includes("-h")) {
    process.stdout.write("Usage: npx github:lanjing-digital/laps-cli install [skills...] [--server URL] [--bin-dir DIR] [--install-dir DIR] [--skills-dir DIR] [--no-skills]\n");
    return 0;
  }
  const options = parseInstallArgs(argumentsList);
  const target = resolveTarget();
  const installDir = path.resolve(options.installDir || defaultInstallDir(target));
  const binDir = path.resolve(options.binDir || defaultBinDir(installDir, target));
  const skillsDir = path.resolve(options.skillsDir || defaultSkillsDir());
  const installedPackage = await installPackage(installDir);
  await ensureBinary(installedPackage);
  const launcherPath = await writeLauncher(installedPackage, binDir, target);
  await writeFile(path.join(installedPackage, installationFile), `${JSON.stringify({ binDir, skillsDir, source: options.source }, null, 2)}\n`, { mode: 0o600 });
  process.stdout.write(`installed laps-cli launcher to ${launcherPath}\n`);
  if (!options.noSkills) {
    for (const skill of options.skills) await installSkill(skill, skillsDir, path.join(installedPackage, "skills"));
  }
  await configureServerDuringInstall(options.server);
  return 0;
}

async function installSkills(argumentsList) {
  const options = parseInstallArgs(argumentsList);
  const skillsDir = path.resolve(options.skillsDir || defaultSkillsDir());
  if (options.noSkills) throw new Error("install-skills cannot use --no-skills");
  for (const skill of options.skills) await installSkill(skill, skillsDir, path.join(packageRoot, "skills"));
  return 0;
}

async function runConfig(argumentsList) {
  if (argumentsList[0] === "set-server") {
    const args = argumentsList.slice(1);
    const value = takeOption(args, "--url", "an http(s) URL");
    if (!value || args.length) throw new Error("Usage: laps-cli config set-server --url https://aps.example.com");
    const baseURL = normalizeServerURL(value);
    await saveSettings(baseURL);
    process.stdout.write(`configured LAPS server: ${baseURL}\nNext: laps-cli auth login\n`);
    return 0;
  }
  if (argumentsList[0] === "show-server") {
    const baseURL = await configuredServerURL();
    if (!baseURL) {
      process.stdout.write("No remote APS server is configured. Run: laps-cli config set-server --url https://aps.example.com\n");
      return 2;
    }
    process.stdout.write(`${baseURL}\n`);
    return 0;
  }
  process.stdout.write("Usage:\n  laps-cli config set-server --url https://aps.example.com\n  laps-cli config show-server\n");
  return argumentsList.includes("--help") || argumentsList.includes("-h") ? 0 : 2;
}

async function installationSettings() {
  const target = resolveTarget();
  const installDir = path.resolve(process.env.LAPS_CLI_INSTALL_DIR || defaultInstallDir(target));
  const recorded = await readJSON(path.join(installDir, installationFile));
  return {
    installDir,
    binDir: path.resolve(recorded.binDir || defaultBinDir(installDir, target)),
    skillsDir: path.resolve(recorded.skillsDir || defaultSkillsDir()),
  };
}

function npxCommand(target = resolveTarget()) {
  return target.goos === "windows" ? "npx.cmd" : "npx";
}

function updateSpec(source) {
  return source === "npm" ? "@lanjing-digital/laps-cli@latest" : `github:${releaseRepository}`;
}

async function runUpdate(argumentsList) {
  if (argumentsList.includes("--help") || argumentsList.includes("-h")) {
    process.stdout.write("Usage: laps-cli update [--source auto|github|npm]\n");
    return 0;
  }
  const { source } = parseUpdateArgs(argumentsList);
  const settings = await installationSettings();
  const sources = source === "auto" ? ["npm", "github"] : [source];
  for (const candidate of sources) {
    if (source === "auto") process.stdout.write(`checking ${candidate} update source...\n`);
    const result = spawnSync(npxCommand(), ["--yes", updateSpec(candidate), "install", "--install-dir", settings.installDir, "--bin-dir", settings.binDir, "--skills-dir", settings.skillsDir, "--source", candidate], {
      stdio: source === "auto" && candidate === "npm" ? "pipe" : "inherit",
    });
    if (!result.error && result.status === 0) {
      process.stdout.write(`laps-cli updated from ${candidate}.\n`);
      return 0;
    }
    if (source !== "auto") throw new Error(`${candidate} update failed; verify Node.js, network access, and the selected source`);
    process.stdout.write(`${candidate} source is unavailable; trying the next source.\n`);
  }
  throw new Error("no update source succeeded; run `laps-cli update --source github` after checking network access");
}

function findBaseURLOption(args) {
  const index = args.indexOf("--base-url");
  if (index >= 0) return args[index + 1] || "";
  const inline = args.find((arg) => arg.startsWith("--base-url="));
  return inline ? inline.slice("--base-url=".length) : "";
}

function isHelpInvocation(args) {
  return args.length === 0 || args.includes("--help") || args.includes("-h") || args[0] === "help";
}

async function requireConfiguredServer(args) {
  if (isHelpInvocation(args)) return args;
  const supplied = findBaseURLOption(args);
  if (supplied) return args;
  const baseURL = await configuredServerURL();
  if (!baseURL) {
    throw new Error("remote APS server is not configured. Run `laps-cli config set-server --url https://aps.example.com` or set SCHEDULING_API_BASE_URL before first use");
  }
  return [...args, "--base-url", baseURL];
}

export async function run(argumentsList) {
  if (argumentsList[0] === "install") return install(argumentsList.slice(1));
  if (argumentsList[0] === "install-skills") return installSkills(argumentsList.slice(1));
  if (argumentsList[0] === "update") return runUpdate(argumentsList.slice(1));
  if (argumentsList[0] === "config") return runConfig(argumentsList.slice(1));
  const binary = await ensureBinary();
  const commandArgs = await requireConfiguredServer(argumentsList);
  const result = spawnSync(binary, commandArgs, { stdio: "inherit" });
  if (result.error) throw result.error;
  return result.status ?? 1;
}
