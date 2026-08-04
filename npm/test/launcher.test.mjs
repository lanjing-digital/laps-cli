import assert from "node:assert/strict";
import test from "node:test";
import { allSkills, normalizeServerURL, parseInstallArgs, parseUpdateArgs, resolveTarget, writeLauncher } from "../lib/launcher.js";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import os from "node:os";
import path from "node:path";

test("maps supported Node targets to Go release assets", () => {
  assert.deepEqual(resolveTarget("darwin", "arm64"), { goos: "darwin", goarch: "arm64", executable: "laps-cli", asset: "laps-cli_darwin_arm64" });
  assert.deepEqual(resolveTarget("win32", "x64"), { goos: "windows", goarch: "amd64", executable: "laps-cli.exe", asset: "laps-cli_windows_amd64.exe" });
});

test("rejects unsupported platform combinations", () => {
  assert.throws(() => resolveTarget("freebsd", "x64"), /unsupported platform/);
  assert.throws(() => resolveTarget("linux", "ia32"), /unsupported platform/);
});

test("install options preserve paths, source, and server configuration", () => {
  assert.deepEqual(parseInstallArgs(["laps-orders", "--bin-dir", "/tmp/bin", "--install-dir", "/tmp/laps", "--skills-dir", "/tmp/skills", "--server", "https://aps.example.com"]), {
    binDir: "/tmp/bin", installDir: "/tmp/laps", skillsDir: "/tmp/skills", server: "https://aps.example.com", source: "github", noSkills: false, skills: ["laps-orders"],
  });
  assert.deepEqual(parseInstallArgs(["--no-skills"]), {
    binDir: undefined, installDir: undefined, skillsDir: undefined, server: undefined, source: "github", noSkills: true, skills: allSkills,
  });
  assert.throws(() => parseInstallArgs(["unknown-skill"]), /unknown skill/);
});

test("normalizes configured APS server URLs and validates update sources", () => {
  assert.equal(normalizeServerURL("https://aps.example.com/"), "https://aps.example.com");
  assert.equal(normalizeServerURL("http://192.168.1.20:3000"), "http://192.168.1.20:3000");
  assert.throws(() => normalizeServerURL("aps.example.com"), /http:\/\//);
  assert.deepEqual(parseUpdateArgs([]), { source: "auto" });
  assert.deepEqual(parseUpdateArgs(["--source", "github"]), { source: "github" });
  assert.throws(() => parseUpdateArgs(["--source", "invalid"]), /auto, npm, or github/);
});

test("writes persistent CLI and MCP launchers without requiring Go", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "laps-launcher-test-"));
  try {
    const target = resolveTarget("darwin", "arm64");
    const cli = await writeLauncher("/opt/laps", directory, target, "laps-cli", "laps-cli.js");
    const mcp = await writeLauncher("/opt/laps", directory, target, "laps-mcp", "laps-mcp.js");
    assert.match(await readFile(cli, "utf8"), /laps-cli\.js/);
    assert.match(await readFile(mcp, "utf8"), /laps-mcp\.js/);
    const windows = await writeLauncher("C:\\laps", directory, resolveTarget("win32", "x64"), "laps-mcp", "laps-mcp.js");
    assert.match(windows, /laps-mcp\.cmd$/);
    assert.match(await readFile(windows, "utf8"), /laps-mcp\.js/);
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});
