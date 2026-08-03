import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { cp, mkdir, readFile, rename, rm, stat, writeFile } from "node:fs/promises";
import { accessSync, constants } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const require = createRequire(import.meta.url);
const packageInfo = require("../../package.json");
const packageRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const skillsRoot = path.join(packageRoot, "skills");
const releaseRepository = process.env.LAPS_CLI_RELEASE_REPOSITORY || "lanjing-digital/laps-cli";

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

function releaseURL(asset) {
  const tag = `v${packageInfo.version}`;
  return `https://github.com/${releaseRepository}/releases/download/${tag}/${asset}`;
}

function cachePath(target) {
  return path.join(packageRoot, "vendor", `${target.goos}-${target.goarch}`, target.executable);
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

export async function ensureBinary() {
  const target = resolveTarget();
  const destination = cachePath(target);
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
  if (target.goos !== "windows") await import("node:fs/promises").then(({ chmod }) => chmod(temporary, 0o755));
  await rename(temporary, destination);
  return destination;
}

function takeOption(args, name) {
  const index = args.indexOf(name);
  if (index < 0) return undefined;
  if (!args[index + 1] || args[index + 1].startsWith("--")) throw new Error(`${name} requires a directory`);
  const value = args[index + 1];
  args.splice(index, 2);
  return value;
}

export function parseInstallArgs(argumentsList) {
  const args = [...argumentsList];
  const binDir = takeOption(args, "--bin-dir");
  const skillsDir = takeOption(args, "--skills-dir");
  const noSkills = args.includes("--no-skills");
  if (noSkills) args.splice(args.indexOf("--no-skills"), 1);
  if (args.some((value) => value.startsWith("--"))) throw new Error(`unknown install option: ${args.find((value) => value.startsWith("--"))}`);
  if (args.some((skill) => !allSkills.includes(skill))) throw new Error(`unknown skill: ${args.find((skill) => !allSkills.includes(skill))}`);
  if (noSkills && args.length > 0) throw new Error("skill names cannot be used with --no-skills");
  return { binDir, skillsDir, noSkills, skills: args.length ? args : allSkills };
}

async function installSkill(skill, targetRoot) {
  const source = path.join(skillsRoot, skill);
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

async function install(argumentsList) {
  if (argumentsList.includes("--help") || argumentsList.includes("-h")) {
    process.stdout.write("Usage: npx @lanjing-digital/laps-cli install [skills...] [--bin-dir DIR] [--skills-dir DIR] [--no-skills]\n");
    return 0;
  }
  const options = parseInstallArgs(argumentsList);
  const binary = await ensureBinary();
  const target = resolveTarget();
  const binDir = options.binDir || (target.goos === "windows" ? path.join(process.env.LOCALAPPDATA || os.homedir(), "laps-cli", "bin") : path.join(os.homedir(), ".local", "bin"));
  const binaryDestination = path.join(binDir, target.executable);
  await mkdir(binDir, { recursive: true });
  await cp(binary, binaryDestination);
  if (target.goos !== "windows") await import("node:fs/promises").then(({ chmod }) => chmod(binaryDestination, 0o755));
  process.stdout.write(`installed laps-cli to ${binaryDestination}\n`);
  if (!options.noSkills) {
    const skillsDir = options.skillsDir || process.env.LAPS_SKILLS_DIR || path.join(os.homedir(), ".agents", "skills");
    for (const skill of options.skills) await installSkill(skill, skillsDir);
  }
  return 0;
}

async function installSkills(argumentsList) {
  const options = parseInstallArgs(argumentsList);
  const skillsDir = options.skillsDir || process.env.LAPS_SKILLS_DIR || path.join(os.homedir(), ".agents", "skills");
  if (options.noSkills) throw new Error("install-skills cannot use --no-skills");
  for (const skill of options.skills) await installSkill(skill, skillsDir);
  return 0;
}

export async function run(argumentsList) {
  if (argumentsList[0] === "install") return install(argumentsList.slice(1));
  if (argumentsList[0] === "install-skills") return installSkills(argumentsList.slice(1));
  const binary = await ensureBinary();
  const result = spawnSync(binary, argumentsList, { stdio: "inherit" });
  if (result.error) throw result.error;
  return result.status ?? 1;
}
