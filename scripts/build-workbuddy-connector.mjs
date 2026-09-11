import { cp, mkdir, readFile, writeFile, mkdtemp, rename, rm, access } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { allSkills, cliVersion, normalizeServerURL } from "../npm/lib/launcher.js";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
export const connectorSkills = allSkills.filter(name => name !== "laps-workbuddy-mcp");

export function connectorConfig(server = "https://lanjingshuzi.cn:3000") {
  const url = normalizeServerURL(server);
  if (new URL(url).protocol !== "https:") throw new Error("WorkBuddy browser authorization requires an HTTPS server");
  // This value is put into a shell command: reject shell metacharacters.
  if (!/^https:\/\/[a-zA-Z0-9.:[\]-]+\/?$/.test(url)) throw new Error("connector default server must be an HTTPS origin without a path");
  const install = `npm install -g @lanjing-digital/laps-cli@${cliVersion} --ignore-scripts --no-audit --no-fund`;
  const init = platform => `${install} && ${platform === "win32" ? "laps-cli.cmd" : "laps-cli"} install --managed --non-interactive --no-skills --default-server ${url}`;
  const commands = suffix => Object.fromEntries(["darwin", "linux", "win32"].map(platform => [platform, `${platform === "win32" ? "laps-cli.cmd" : "laps-cli"} ${suffix}`]));
  return {
    runtime: { type: "node", version: "20" },
    npmRegistry: "https://registry.npmjs.org",
    init: Object.fromEntries(["darwin", "linux", "win32"].map(platform => [platform, init(platform)])),
    auth: commands("auth login --no-browser"),
    status: commands("auth status --local"),
    unAuth: commands("auth logout"),
    statusMatch: '"authenticated"\\s*:\\s*true',
    authWaitForExit: true,
    env: { LAPS_CLI_SKILLS_VERSION: cliVersion },
  };
}

export async function buildConnector(destination, server) {
  const config = connectorConfig(server);
  const parent = path.dirname(path.resolve(destination));
  await mkdir(parent, { recursive: true });
  // The bundle is generated; replacing a non-bundle directory is forbidden.
  let exists = false;
  try { await access(destination); exists = true; } catch (error) { if (error.code !== "ENOENT") throw error; }
  if (exists) {
    const previous = JSON.parse(await readFile(path.join(destination, "connector-meta.json"), "utf8"));
    if (previous.source !== "lanjing-laps-cli") throw new Error("destination is not a LAPS connector bundle");
  }
  const staging = await mkdtemp(path.join(parent, ".laps-connector-"));
  const bundle = path.join(staging, "bundle");
  try {
    await mkdir(path.join(bundle, "skills"), { recursive: true });
    for (const skill of connectorSkills) {
      await cp(path.join(root, "skills", skill), path.join(bundle, "skills", skill), { recursive: true });
      await writeFile(path.join(bundle, "skills", skill, ".laps-version.json"), JSON.stringify({ version: cliVersion }) + "\n");
    }
    const meta = {
      name: "LAPS 智能排产", name_zh: "LAPS 智能排产", name_en: "LAPS Production Planning",
      description: "通过自然语言管理生产订单、物料齐套、产能和排产计划，先试算比较方案，再确认提交。",
      description_zh: "通过自然语言管理生产订单、物料齐套、产能和排产计划，先试算比较方案，再确认提交。",
      description_en: "Manage production orders, materials, capacity and schedules. Preview candidate plans before confirming changes.",
      source: "lanjing-laps-cli", type: "cli", version: cliVersion, minWorkbuddyVersion: "4.24.0",
      examples_zh: ["查看当前待排订单", "检查本周订单是否齐套", "试排待生产订单并展示甘特图"],
      examples_en: ["Show pending production orders", "Check material readiness for this week", "Preview a production schedule and show its Gantt chart"],
    };
    await writeFile(path.join(bundle, "connector-meta.json"), JSON.stringify(meta, null, 2) + "\n");
    await writeFile(path.join(bundle, "cli.json"), JSON.stringify(config, null, 2) + "\n");
    await writeFile(path.join(bundle, "icon.svg"), '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" role="img" aria-label="LAPS"><rect x="4" y="4" width="56" height="56" rx="12" fill="#2563eb"/><path d="M16 18v28h34" fill="none" stroke="#eff6ff" stroke-width="3"/><path d="M22 22h17M29 31h19M22 40h10" stroke="#eff6ff" stroke-width="6" stroke-linecap="round"/></svg>\n');
    await writeFile(path.join(bundle, "README.md"), `# LAPS WorkBuddy CLI 连接器\n\n版本 ${cliVersion}。将此目录整体打包提交 WorkBuddy 连接器审核，接入方式选择 CLI + Skill。\n\n复用公开 npm 包 @lanjing-digital/laps-cli@${cliVersion} 与对应的 GitHub 六平台二进制；目录内没有另一份 CLI 实现，也不包含 MCP 配置。八份业务技能由主 CLI 技能源生成。\n\n## 地址与登录\n\n首次安装默认使用 ${server || "https://lanjingshuzi.cn:3000"}（3000 通用演示版），已设置的地址会保留。用户可在 WorkBuddy 中要求设置企业 APS 地址，执行：\n\n\x60\x60\x60sh\nlaps-cli config set-server --url https://aps.example.com\nlaps-cli auth login --no-browser\n\x60\x60\x60\n\n修改地址后重新连接，凭据必须属于该地址。3001 派逊和 3003 世净是独立实例，须由用户明确选择。\n\nWorkBuddy 准备 Node.js 20 并管理 npm 安装路径，用户无需预装 Node.js 或 Go。安装不读 stdin、不打开登录页。点击连接后由 WorkBuddy 打开 CLI 输出的授权地址；authWaitForExit 保持本地回调进程，登录完成后凭据由同一 laps-cli 持久化。status 使用 auth status --local，只读取可续期的本地会话，不联网、不刷新、不写文件；远端撤销由下次业务请求检测。取消连接执行 auth logout 撤销远端会话并清理本地凭据。\n\n## 版本一致性\n\n安装锁定版本 ${cliVersion}，随包技能也为此版本。普通命令可提示新 CLI 版本和 Skills 不一致；当连接器内的技能落后时应更新整个 WorkBuddy 连接器。连接器版本由 cli/VERSION 和 package.json 一起控制，重新运行 cli/scripts/build-workbuddy-connector.mjs 生成分发目录。\n\n## 提交与验收\n\n规范：https://open.workbuddy.cn/docs/connector#cli-skill-接入\n\n提交前确认 npm 对应版本与 GitHub Release 已公开；在 WorkBuddy 中验收安装、连接、重启后状态恢复、取消连接及重新连接。命令级自动测试不能代替市场审核和客户端实际验收。\n`);
    const backup = path.join(staging, "previous");
    let backedUp = false;
    try { await rename(destination, backup); backedUp = true; } catch (error) { if (error.code !== "ENOENT") throw error; }
    try { await rename(bundle, destination); } catch (error) {
      if (backedUp) await rename(backup, destination);
      throw error;
    }
  } finally { await rm(staging, { recursive: true, force: true }); }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const destination = path.resolve(process.argv[2] || path.join(root, "dist", "laps-workbuddy-cli"));
  await buildConnector(destination, process.argv[3]);
  console.log(destination);
}
