import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { createRequire } from "node:module";
import { copy } from "@edupilot/core/native/strings";

test("the installed app and both language headers identify EduPilot OS", () => {
  const manifest = JSON.parse(
    readFileSync(createRequire(import.meta.url).resolve("#manifest"), "utf8"),
  );
  assert.equal(manifest.expo.name, "EduPilot OS");
  for (const locale of Object.values(copy)) {
    assert.equal(locale.brand, manifest.expo.name);
  }
});
