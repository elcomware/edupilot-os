import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import type { Session } from "@edupilot/api-client";
import { api, useAuth } from "@edupilot/core/web/app/session";
import {
  useLocale,
  usePreferences,
} from "@edupilot/core/web/app/preferences";
import { useRoute } from "@edupilot/core/web/app/router";
import { t } from "@edupilot/core/web/app/copy";
import { errorKey } from "@edupilot/core/web/app/errors";
import { directoryIcons, LoginPage, workspacePages } from "./pages";
import { Directory } from "@edupilot/core/web/features/directory/Directory";
import { PreferencesPage } from "@edupilot/core/web/features/preferences/PreferencesPage";
import { TenantAdminLayout } from "@edupilot/core/web/layouts/TenantAdminLayout";
import { StatePanel } from "@edupilot/core/web/components/primitives";
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
  const WorkspacePage =
    route.page === "module" ? workspacePages.get(route.key) : undefined;
  let content;
  if (route.page === "settings")
    content = (
      <PreferencesPage
        modules={allowed ? (modules.data?.items ?? []) : []}
        catalogStatus={
          !allowed || modules.isError
            ? "unavailable"
            : modules.isPending
              ? "loading"
              : "ready"
        }
      />
    );
  else if (
    route.page === "unknown" ||
    (route.page === "module" && !WorkspacePage)
  )
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
  else if (route.page === "module" && WorkspacePage)
    content = modules.data.items.some((m) => m.key === route.key) ? (
      <WorkspacePage
        key={route.key}
        moduleKey={route.key}
        principal={principal}
      />
    ) : (
      <StatePanel type="error" title={t(l, "unavailable")} />
    );
  else content = <Directory modules={modules.data.items} icons={directoryIcons} />;
  return (
    <TenantAdminLayout
      principal={principal}
      modules={modules.data?.items ?? []}
    >
      {content}
    </TenantAdminLayout>
  );
}
export function App() {
  const session = useAuth((s) => s.session);
  return (
    <>
      <PreferencesEffect />
      {session ? <SignedIn session={session} /> : <LoginPage />}
    </>
  );
}
