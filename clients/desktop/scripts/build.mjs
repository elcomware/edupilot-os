// Input: the existing web build. Output: a native binary with those bytes embedded.
// Invoke after npm run build --workspace @edupilot/web (from repository root).
import { cp, mkdir, readFile, rm } from "node:fs/promises";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, resolve, join } from "node:path";

const desktop = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const webDist = resolve(desktop, "../web/dist");
const dest = resolve(desktop, "frontend/dist");
const args = process.argv.slice(2);
if (args.length > 1 || (args.length && args[0] !== "--assets-only"))
  throw new Error(
    "Usage: node clients/desktop/scripts/build.mjs [--assets-only]",
  );
try {
  await readFile(join(webDist, "index.html"));
} catch {
  throw new Error(
    "Build clients/web first: npm run build --workspace @edupilot/web",
  );
}
await rm(dest, { recursive: true, force: true });
await cp(webDist, dest, { recursive: true });
if (!args.includes("--assets-only")) {
  const bin = resolve(
    desktop,
    "bin",
    process.platform === "win32" ? "edupilot.exe" : "edupilot",
  );
  await mkdir(dirname(bin), { recursive: true });
  const flags = ["build", "-trimpath", "-tags", "production"];
  if (process.platform === "win32") flags.push("-ldflags=-H windowsgui");
  const p = spawnSync("go", [...flags, "-o", bin, "."], {
    cwd: desktop,
    stdio: "inherit",
  });
  if (p.error) throw p.error;
  if (p.status !== 0) process.exit(p.status ?? 1);
}
