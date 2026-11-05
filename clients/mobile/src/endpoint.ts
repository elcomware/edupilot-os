/** Return null for an unsafe/missing deployment address. Permit plain HTTP only
 * for simulator loopback endpoints in a dev bundle. Never accept login-time URLs. */
export function apiRoot(
  raw: string | undefined,
  development: boolean,
): string | null {
  if (!raw) return null;
  try {
    const url = new URL(raw);
    if (
      !url.hostname ||
      url.username ||
      url.password ||
      url.pathname !== "/" ||
      url.search ||
      url.hash ||
      /[?#]/.test(raw)
    )
      return null;
    const simulator = ["localhost", "127.0.0.1", "[::1]", "10.0.2.2"].includes(
      url.hostname,
    );
    if (
      url.protocol !== "https:" &&
      !(development && simulator && url.protocol === "http:")
    )
      return null;
    return url.origin + "/api/v1";
  } catch {
    return null;
  }
}
