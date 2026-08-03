#!/usr/bin/env node

import { run } from "../lib/launcher.js";

try {
  const exitCode = await run(process.argv.slice(2));
  if (Number.isInteger(exitCode)) process.exitCode = exitCode;
} catch (error) {
  process.stderr.write(`laps-cli: ${error.message}\n`);
  process.exitCode = 1;
}
