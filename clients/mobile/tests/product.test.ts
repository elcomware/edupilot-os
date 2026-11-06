import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { URL } from "node:url";
import { copy } from "../../../resources/native/strings.ts";

test("the installed app and both language headers identify EduPilot OS", () => {
  const manifest = JSON.parse(
    readFileSync(new URL("../app.json", import.meta.url), "utf8"),
  );
  assert.equal(manifest.expo.name, "EduPilot OS");
  for (const locale of Object.values(copy)) {
    assert.equal(locale.brand, manifest.expo.name);
  }
});
