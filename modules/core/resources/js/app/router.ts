import { useEffect, useSyncExternalStore } from "react";
type Route =
  | { page: "workspace" | "settings" | "unknown" }
  | { page: "module"; key: string };
export function readRoute(hash: string): Route {
  if (hash === "" || hash === "#/" || hash === "#/workspace")
    return { page: "workspace" };
  if (hash === "#/settings") return { page: "settings" };
  const match = /^#\/modules\/([a-z0-9][a-z0-9_-]*)$/.exec(hash);
  return match ? { page: "module", key: match[1] } : { page: "unknown" };
}
const subscribe = (cb: () => void) => {
  window.addEventListener("hashchange", cb);
  return () => window.removeEventListener("hashchange", cb);
};
export function useRoute() {
  return readRoute(
    useSyncExternalStore(
      subscribe,
      () => window.location.hash,
      () => "",
    ),
  );
}
export const moduleHref = (key: string) =>
  "#/modules/" + encodeURIComponent(key);

// Hash routes can also change through browser history while a modal is open.
// Capture the transition before React observes it, restoring the previous
// location when the user keeps the draft. No form data is written to disk.
export function useUnsavedGuard(dirty: boolean, warning: string) {
  useEffect(() => {
    if (!dirty) return;
    let accepted = window.location.href;
    const unload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = "";
    };
    const hash = (e: HashChangeEvent) => {
      if (window.confirm(warning)) {
        accepted = location.href;
        return;
      }
      e.stopImmediatePropagation();
      window.history.pushState(null, "", accepted);
    };
    window.addEventListener("beforeunload", unload);
    window.addEventListener("hashchange", hash, true);
    return () => {
      window.removeEventListener("beforeunload", unload);
      window.removeEventListener("hashchange", hash, true);
    };
  }, [dirty, warning]);
}
