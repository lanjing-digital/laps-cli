import assert from "node:assert/strict";
import test from "node:test";
import { mkdtemp, mkdir, readFile, readdir, writeFile, rm } from "node:fs/promises";
import { createHash } from "node:crypto";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { allSkills, cliVersion, ensureBinary, resolveTarget, installSkill, parseInstallArgs, parseUpdateArgs, updateInstallArguments, packageManagerInvocation } from "../lib/launcher.js";
import { buildConnector, connectorConfig, connectorSkills } from "../../scripts/build-workbuddy-connector.mjs";

const root = fileURLToPath(new URL("../../", import.meta.url));

test("release version and all nine source skills match the installer", async () => {
  assert.equal((await readFile(path.join(root,"VERSION"),"utf8")).trim(),cliVersion);
  const dirs=await readdir(path.join(root,"skills"),{withFileTypes:true});
  assert.deepEqual(allSkills.toSorted(),dirs.filter(d=>d.isDirectory()).map(d=>d.name).toSorted());
});

test("explicit unattended options and update selection are preserved", () => {
  const options=parseInstallArgs(["--non-interactive","--managed","--no-skills","--default-server","https://aps.example.com"]);
  assert.equal(options.nonInteractive,true); assert.equal(options.managed,true);
  assert.equal(options.defaultServer,"https://aps.example.com");
  assert.throws(()=>parseInstallArgs(["--managed","--bin-dir","/tmp/bin"]),/runtime/);
  assert.throws(()=>parseInstallArgs(["--server","https://a.test","--default-server","https://b.test"]),/mutually exclusive/);
  assert.throws(()=>parseUpdateArgs(["--check","--force"]),/mutually exclusive/);
  assert.deepEqual(parseUpdateArgs(["--check","--json"]),{source:"auto",check:true,json:true,force:false});
  const settings={installDir:"/tmp/cli",binDir:"/tmp/bin",skillsDir:"/tmp/skills",skills:[]};
  assert.ok(updateInstallArguments("npm",settings,"1.2.3").includes("@lanjing-digital/laps-cli@1.2.3"));
  assert.ok(updateInstallArguments("npm",settings,"1.2.3").includes("--no-skills"));
  settings.skills=["laps-orders"];
  assert.equal(updateInstallArguments("github",settings,"1.2.3").at(-1),"laps-orders");
});

test("Windows managed npm uses Node and keeps shell-sensitive arguments literal", () => {
  const args=["install","--prefix","C:\\Users\\Name & Co\\runtime"];
  assert.deepEqual(packageManagerInvocation("npm",args,"win32","C:\\Managed Node\\node.exe",{}),{
    command:"C:\\Managed Node\\node.exe",args:["C:\\Managed Node\\node_modules\\npm\\bin\\npm-cli.js",...args],
  });
  assert.deepEqual(packageManagerInvocation("npx",["--yes"],"win32","C:\\Node\\node.exe",{npm_execpath:"C:\\npm\\bin\\npm-cli.js"}),{
    command:"C:\\Node\\node.exe",args:["C:\\npm\\bin\\npx-cli.js","--yes"],
  });
});

test("clean binary install downloads verified bytes and never runs a compiler", async () => {
  const directory=await mkdtemp(path.join(os.tmpdir(),"laps-binary-test-"));
  const oldFetch=globalThis.fetch;
  const binary=Buffer.from("fixture precompiled binary");
  const hash=createHash("sha256").update(binary).digest("hex");
  const target=resolveTarget(); let calls=0;
  globalThis.fetch=async url=>{ calls++; return new Response(String(url).endsWith("checksums.txt") ? `${hash}  ${target.asset}\n` : binary); };
  try {
    const executable=await ensureBinary(directory);
    assert.deepEqual(await readFile(executable),binary);
    assert.equal(await ensureBinary(directory),executable);
    assert.equal(calls,2);
    const invalid=path.join(directory,"invalid");
    globalThis.fetch=async url=>new Response(String(url).endsWith("checksums.txt") ? `${"0".repeat(64)}  ${target.asset}\n` : binary);
    await assert.rejects(ensureBinary(invalid),/checksum verification failed/);
  } finally { globalThis.fetch=oldFetch; await rm(directory,{recursive:true,force:true}); }
});

test("skill install records its version and only replaces the selected skill", async () => {
  const directory=await mkdtemp(path.join(os.tmpdir(),"laps-skills-test-"));
  try {
    const unrelated=path.join(directory,"unrelated"); await mkdir(unrelated); await writeFile(path.join(unrelated,"SKILL.md"),"preserve");
    const log={write(){}};
    await installSkill("laps-orders",directory,path.join(root,"skills"),"0.1.0",log);
    await installSkill("laps-orders",directory,path.join(root,"skills"),cliVersion,log);
    assert.equal(JSON.parse(await readFile(path.join(directory,"laps-orders/.laps-version.json"),"utf8")).version,cliVersion);
    assert.equal(await readFile(path.join(unrelated,"SKILL.md"),"utf8"),"preserve");
    assert.deepEqual((await readdir(directory)).sort(),["laps-orders","unrelated"]);
  } finally { await rm(directory,{recursive:true,force:true}); }
});

test("WorkBuddy CLI bundle uses the same published package and exact source skills", async () => {
  const directory=await mkdtemp(path.join(os.tmpdir(),"laps-connector-test-"));
  try {
    const destination=path.join(directory,"bundle"); await buildConnector(destination);
    const meta=JSON.parse(await readFile(path.join(destination,"connector-meta.json"),"utf8"));
    const config=JSON.parse(await readFile(path.join(destination,"cli.json"),"utf8"));
    assert.equal(meta.type,"cli"); assert.equal(meta.version,cliVersion); assert.equal(config.authWaitForExit,true);
    for (const platform of ["darwin","linux","win32"]) {
      assert.ok(config.init[platform].includes(`@lanjing-digital/laps-cli@${cliVersion}`));
      assert.ok(config.init[platform].includes("--managed --non-interactive --no-skills --default-server https://lanjingshuzi.cn:3000"));
      assert.ok(config.auth[platform].endsWith("auth login --no-browser"));
      assert.ok(config.status[platform].endsWith("auth status --local"));
      assert.ok(config.unAuth[platform].endsWith("auth logout"));
    }
    assert.match('{"authenticated": true}',new RegExp(config.statusMatch));
    assert.doesNotMatch('{"authenticated": false}',new RegExp(config.statusMatch));
    assert.equal(connectorSkills.length,8);
    for (const name of connectorSkills) for (const file of ["SKILL.md","references/commands.md"]) {
      assert.equal(await readFile(path.join(destination,"skills",name,file),"utf8"),await readFile(path.join(root,"skills",name,file),"utf8"));
    }
    assert.ok(!(await readdir(destination)).includes("mcp.json"));
    assert.throws(()=>connectorConfig("https://example.com/;evil"),/without a path/);
    await buildConnector(destination,"https://aps.example.com");
    assert.match(await readFile(path.join(destination,"cli.json"),"utf8"),/https:\/\/aps.example.com/);
  } finally { await rm(directory,{recursive:true,force:true}); }
});

test("version and install help work without a server or Go", () => {
  const run=args=>spawnSync(process.execPath,[path.join(root,"npm/bin/laps-cli.js"),...args],{encoding:"utf8",env:{...process.env,SCHEDULING_API_BASE_URL:""}});
  const version=run(["--version"]); assert.equal(version.status,0); assert.equal(version.stdout.trim(),cliVersion);
  const help=run(["install","--help"]); assert.equal(help.status,0); assert.match(help.stdout,/--non-interactive/);
});
