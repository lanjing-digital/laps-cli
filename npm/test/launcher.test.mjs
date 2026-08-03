import assert from "node:assert/strict";
import test from "node:test";
import { allSkills, parseInstallArgs, resolveTarget } from "../lib/launcher.js";

test("maps supported Node targets to Go release assets", () => {
  assert.deepEqual(resolveTarget("darwin", "arm64"), { goos: "darwin", goarch: "arm64", executable: "laps-cli", asset: "laps-cli_darwin_arm64" });
  assert.deepEqual(resolveTarget("win32", "x64"), { goos: "windows", goarch: "amd64", executable: "laps-cli.exe", asset: "laps-cli_windows_amd64.exe" });
});

test("rejects unsupported platform combinations", () => {
  assert.throws(() => resolveTarget("freebsd", "x64"), /unsupported platform/);
  assert.throws(() => resolveTarget("linux", "ia32"), /unsupported platform/);
});

test("install options keep falsey controls and validate skills", () => {
  assert.deepEqual(parseInstallArgs(["laps-orders", "--bin-dir", "/tmp/bin", "--skills-dir", "/tmp/skills"]), {
    binDir: "/tmp/bin", skillsDir: "/tmp/skills", noSkills: false, skills: ["laps-orders"],
  });
  assert.deepEqual(parseInstallArgs(["--no-skills"]), {
    binDir: undefined, skillsDir: undefined, noSkills: true, skills: allSkills,
  });
  assert.throws(() => parseInstallArgs(["unknown-skill"]), /unknown skill/);
});
