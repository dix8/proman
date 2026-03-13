import { spawnSync } from "node:child_process";
import path from "node:path";

const cwd = process.cwd();

const scripts = [
  "smoke-auth.mjs",
  "smoke-projects.mjs",
  "smoke-versions.mjs",
  "smoke-announcements.mjs",
  "smoke-markdown-preview.mjs",
  "smoke-compare-export.mjs",
];

const summary = [];
let hasFailure = false;

for (const script of scripts) {
  const child = spawnSync(process.execPath, [path.join("scripts", script)], {
    cwd,
    env: process.env,
    encoding: "utf8",
  });

  let parsed = null;
  let parseError = "";
  if (child.stdout) {
    try {
      parsed = JSON.parse(child.stdout.trim());
    } catch (error) {
      parseError = String(error);
    }
  }

  const failedChecks = Array.isArray(parsed)
    ? parsed.filter((item) => !item.ok)
    : [];
  const scriptResult = {
    script,
    exitCode: child.status ?? 0,
    parsed: Array.isArray(parsed),
    failedChecks: failedChecks.length,
    parseError,
    stderr: child.stderr?.trim() || "",
    results: parsed,
  };

  if (
    scriptResult.exitCode !== 0 ||
    scriptResult.failedChecks > 0 ||
    scriptResult.parseError
  ) {
    hasFailure = true;
  }

  summary.push(scriptResult);
}

console.log(JSON.stringify(summary, null, 2));

if (hasFailure) {
  process.exit(1);
}
