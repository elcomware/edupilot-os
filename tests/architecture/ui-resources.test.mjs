import assert from "node:assert/strict";
import { existsSync, globSync, readFileSync } from "node:fs";
import test from "node:test";

test("UI imports resolve through their owning module or a sibling file", () => {
  assert.ok(existsSync("modules/core/resources/package.json"));
  assert.ok(!existsSync("resources/js") && !existsSync("resources/css") && !existsSync("resources/native"));
  const sources = globSync([
    "modules/*/resources/{js,native}/**/*.{ts,tsx}",
    "clients/{web,mobile}/{src,tests}/**/*.{ts,tsx}",
    "clients/mobile/App.tsx",
  ]);
  assert.ok(sources.length > 0);
  for (const path of sources) {
    const source = readFileSync(path, "utf8");
    const imports = source.matchAll(/(?:\bfrom\s*|\bimport\s*(?:\(\s*)?|\brequire\(\s*)["']([^"']+)["']/g);
    for (const [, name] of imports) {
      assert.ok(!name.startsWith("../"), path + " traverses a parent to import " + name);
      const owner = path.split("/");
      if (owner[0] === "modules" && name.startsWith("@edupilot/")) {
        const dependency = name.split("/")[1];
        assert.ok([owner[1], "core", "api-client"].includes(dependency),
          path + " imports another feature directly: " + name);
      }
    }
  }
});
