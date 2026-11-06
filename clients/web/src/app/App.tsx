import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import type { Session } from "@edupilot/api-client";
import { api, useAuth } from "./session";
import { useLocale, usePreferences } from "./preferences";
import { useRoute } from "./router";
import { t } from "./copy";
import { errorKey } from "./errors";
import { Login } from "../../../../modules/auth/resources/js/pages/Login";
import { Directory } from "../features/directory/Directory";
import { WorkspacePage } from "../features/collections/WorkspacePage";
import { PreferencesPage } from "../features/preferences/PreferencesPage";
import { Shell } from "../shell/Shell";
import { StatePanel } from "../components/primitives";
function PreferencesEffect() {
  const prefs = usePreferences((s) => s.value);
  useEffect(() => {
    const root = document.documentElement,
      media = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      root.lang = prefs.locale;
      root.dataset.theme =
        prefs.appearance === "system"
          ? media.matches
            ? "dark"
            : "light"
          : prefs.appearance;
      root.dataset.accent = prefs.accent;
      root.dataset.density = prefs.density;
      root.dataset.width = prefs.width;
    };
    apply();
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, [prefs]);
  return null;
}
function SignedIn({ session }: { session: Session }) {
  const route = useRoute(),
    l = useLocale();
  const who = useQuery({
    queryKey: ["principal", session.principal.school, session.principal.id],
    queryFn: ({ signal }) => api.session(signal),
    initialData: session.principal,
    initialDataUpdatedAt: 0,
  });
  const principal = who.data,
    allowed = principal.permissions.includes("modules.read");
  const modules = useQuery({
    queryKey: ["modules", principal.school, principal.id],
    queryFn: ({ signal }) => api.modules(signal),
    enabled: allowed,
  });
  let content;
  if (route.page === "settings") content = <PreferencesPage
    modules={allowed ? modules.data?.items ?? [] : []}
    catalogStatus={!allowed || modules.isError ? "unavailable" : modules.isPending ? "loading" : "ready"}
  />;
  else if (route.page === "unknown")
    content = (
      <StatePanel
        type="error"
        title={t(l, "unavailable")}
        action={
          <a href="#/workspace" className="btn">
            {t(l, "back")}
          </a>
        }
      />
    );
  else if (!allowed)
    content = <StatePanel type="error" title={t(l, "denied")} />;
  else if (route.page === "module")
    content = (
      <WorkspacePage
        key={route.key}
        moduleKey={route.key}
        principal={principal}
      />
    );
  else if (modules.isPending)
    content = <StatePanel type="busy" title={t(l, "loading")} />;
  else if (modules.isError)
    content = (
      <StatePanel
        type="error"
        title={t(l, "loadFailed")}
        body={t(l, errorKey(modules.error))}
        action={
          <button className="btn" onClick={() => void modules.refetch()}>
            {t(l, "retry")}
          </button>
        }
      />
    );
  else content = <Directory modules={modules.data.items} />;
  return (
    <Shell principal={principal} modules={modules.data?.items ?? []}>
      {content}
    </Shell>
  );
}
export function App() {
  const session = useAuth((s) => s.session);
  return (
    <>
      <PreferencesEffect />
      {session ? <SignedIn session={session} /> : <Login />}
    </>
  );
}
