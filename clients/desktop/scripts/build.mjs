// Input: the existing web build. Output: a native binary with those bytes embedded.
// Invoke after npm run build --workspace @edupilot/web (from repository root).
import { cp, mkdir, readFile, rm } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import { createRequire } from "node:module";
import { dirname, resolve, join } from "node:path";

const locate = createRequire(import.meta.url).resolve;
const webDist = join(dirname(locate("@edupilot/web/package.json")), "dist");
const desktop = dirname(import.meta.dirname);
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
await cp(locate("@edupilot/core/brand/png/edupilot-app-icon-256.png"), join(dest, "desktop-icon.png"));
if (!args.includes("--assets-only")) {
  const target = process.env.GOOS ?? (process.platform === "win32" ? "windows" : process.platform);
  const windows = target === "windows";
  if (windows) {
    // The resource compiler runs on the build host, including when cross-compiling.
    const { GOOS, GOARCH, ...hostEnv } = process.env;
    const arch = GOARCH || execFileSync("go", ["env", "GOARCH"], { encoding: "utf8" }).trim();
    execFileSync("go", ["tool", "rsrc", "-arch", arch, "-ico",
      locate("@edupilot/core/brand/platform/edupilot-app-icon.ico"),
      "-o", "icon_windows_" + arch + ".syso"], { cwd: desktop, env: hostEnv, stdio: "inherit" });
  }
  const bin = join(desktop, "bin", windows ? "edupilot.exe" : "edupilot");
  await mkdir(dirname(bin), { recursive: true });
  const flags = ["build", "-trimpath", "-tags", "production"];
  if (windows) flags.push("-ldflags=-H windowsgui");
  execFileSync("go", [...flags, "-o", bin, "."], { cwd: desktop, stdio: "inherit" });
}
