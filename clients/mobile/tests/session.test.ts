import test from "node:test";
import assert from "node:assert/strict";
import type { Credentials, Principal } from "@edupilot/api-client";
import { SessionModel, type TokenStore } from "../../../resources/native/session.ts";

const principal: Principal = {
  id: "fixture-user",
  school: "school-A",
  permissions: ["workspace.read"],
};
const credentials: Credentials = {
  school: "school-A",
  email: "test@example.test",
  password: "fixture",
};
function setup(cached: string | null, status = 200, logoutStatus = 204) {
  let saved = cached,
    logouts = 0,
    presented: string | null = null;
  const store: TokenStore = {
    read: async () => saved,
    write: async (t) => {
      saved = t;
    },
    remove: async () => {
      saved = null;
    },
  };
  const model = new SessionModel(store, (getToken) => ({
    session: async () => {
      presented = getToken();
      if (status !== 200) throw { status };
      return principal;
    },
    login: async () => ({ principal, token: "new-token" }),
    logout: async () => {
      logouts++;
      presented = getToken();
      if (logoutStatus !== 204) throw { status: logoutStatus };
    },
  }));
  return {
    model,
    store,
    saved: () => saved,
    logouts: () => logouts,
    presented: () => presented,
  };
}
test("restores only a server-validated identity", async () => {
  const c = setup("cached");
  await c.model.restore();
  assert.equal(c.presented(), "cached");
  assert.deepEqual(c.model.snapshot(), { kind: "member", principal });
});
test("network failure locks private UI but keeps the record for retry", async () => {
  const c = setup("cached", 503);
  await c.model.restore();
  assert.equal(c.model.snapshot().kind, "retry");
  assert.equal(c.model.getToken(), null);
  assert.equal(c.saved(), "cached");
});
test("expired credential is forgotten without rendering private content", async () => {
  const c = setup("expired", 401);
  await c.model.restore();
  assert.deepEqual(c.model.snapshot(), { kind: "guest", reason: "expired" });
  assert.equal(c.saved(), null);
});
test("logout revokes the active bearer and clears local storage", async () => {
  const c = setup(null);
  await c.model.restore();
  await c.model.login(credentials);
  assert.equal(c.saved(), "new-token");
  await c.model.logout();
  assert.equal(c.presented(), "new-token");
  assert.equal(c.logouts(), 1);
  assert.equal(c.model.getToken(), null);
  assert.equal(c.saved(), null);
  assert.equal(c.model.snapshot().kind, "guest");
});
test("failed revocation still signs out locally and reports the limit", async () => {
  const c = setup("cached", 200, 503);
  await c.model.restore();
  await c.model.logout();
  assert.equal(c.saved(), null);
  assert.deepEqual(c.model.snapshot(), {
    kind: "guest",
    reason: "remoteLogout",
  });
});
test("locked secure storage prevents resuming private content after logout", async () => {
  const c = setup("cached");
  await c.model.restore();
  c.store.remove = async () => {
    throw new Error("locked");
  };
  await c.model.logout();
  assert.equal(c.model.getToken(), null);
  assert.equal(c.model.snapshot().kind, "erase");
});
test("a login whose token cannot be saved revokes rather than publishing the member", async () => {
  const c = setup(null);
  await c.model.restore();
  c.store.write = async () => {
    throw new Error("locked");
  };
  await c.model.login(credentials);
  assert.equal(c.logouts(), 1);
  assert.equal(c.model.getToken(), null);
  assert.deepEqual(c.model.snapshot(), { kind: "guest", reason: "storage" });
});
