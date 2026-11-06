import { expect, test } from "@playwright/test";
import { account, session } from "./accounts";

test("school grants protect the catalog and isolate school work lists", async ({
  request,
}) => {
  const a = account("A"),
    b = account("B");
  const owner = await session(request, a),
    other = await session(request, b);
  const base = "/api/v1/modules/tenancy/workspace";
  const catalog = await request.get("/api/v1/modules", { headers: owner });
  expect(catalog.status()).toBe(200);
  expect(
    (await catalog.json()).items.map((m: { key: string }) => m.key),
  ).toEqual(["tenancy", "auth"]);
  expect(
    (
      await request.get("/api/v1/modules", {
        headers: { ...owner, "X-School-ID": b.school },
      })
    ).status(),
  ).toBe(403);
  const created = await request.post(base, {
    headers: owner,
    data: { title: "Campus review" },
  });
  expect(created.status()).toBe(201);
  const item = await created.json(),
    path = base + "/" + item.id;
  expect((await request.get(path, { headers: owner })).status()).toBe(200);
  expect((await request.get(path, { headers: other })).status()).toBe(404);
  expect(
    (
      await request.patch(path, {
        headers: other,
        data: { title: "Changed", status: "closed", version: 1 },
      })
    ).status(),
  ).toBe(404);
  expect(
    (await request.get("/api/v1/students", { headers: owner })).status(),
  ).toBe(404);
  expect(
    (await request.delete("/api/v1/session", { headers: owner })).status(),
  ).toBe(204);
  expect((await request.get(path, { headers: owner })).status()).toBe(401);
});
