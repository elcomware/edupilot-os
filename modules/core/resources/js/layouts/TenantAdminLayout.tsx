import {AdminLayout} from "./AdminLayout";
import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  BookOpen,
  LayoutGrid,
  LogOut,
  Menu,
  Pin,
  Settings2,
  X,
} from "lucide-react";
import type { ModuleDescriptor, Principal } from "@edupilot/api-client";
import { signOut } from "@edupilot/core/web/app/session";
import { t } from "@edupilot/core/web/app/copy";
import { orderedPinnedModules, useLocale, usePreferences } from "@edupilot/core/web/app/preferences";
import { moduleHref, useRoute } from "@edupilot/core/web/app/router";
import { Brand } from "@edupilot/core/web/components/primitives";
export function TenantAdminLayout({
  principal,
  modules,
  children,
}: {
  principal: Principal;
  modules: ModuleDescriptor[];
  children: ReactNode;
}) {
  const l = useLocale(),
    pins = usePreferences((s) => s.value.pins),
    route = useRoute();
  const [open, setOpen] = useState(false),
    main = useRef<HTMLElement>(null);
  const pinned = orderedPinnedModules(modules, pins),
    current =
      route.page === "module" ? modules.find((m) => m.key === route.key) : null;
  const pageName =
    route.page === "settings"
      ? t(l, "prefs")
      : (current?.title[l] ?? t(l, "directory"));
  useEffect(() => {
    const cb = () => {
      setOpen(false);
      main.current?.focus();
    };
    window.addEventListener("hashchange", cb);
    return () => window.removeEventListener("hashchange", cb);
  }, []);
  useEffect(() => {
    document.title = pageName + " · EduPilot OS";
  }, [pageName]);
  return (
    <AdminLayout scope={principal.school} title={pageName} mainRef={main} navigation={<>
      <a
        href="#content"
        className="skip"
        onClick={(e) => {
          e.preventDefault();
          main.current?.focus();
        }}
      >
        {t(l, "skip")}
      </a>
      <aside className="rail">
        <div className="rail-top">
          <a href="#/workspace" className="home-brand">
            <Brand />
          </a>
          <button
            type="button"
            className="rail-toggle icon-button"
            aria-expanded={open}
            aria-controls="school-nav"
            aria-label={t(l, "menu")}
            onClick={() => setOpen(!open)}
          >
            {open ? <X /> : <Menu />}
          </button>
        </div>
        <div id="school-nav" className={"rail-body" + (open ? " is-open" : "")}>
          <nav aria-label={t(l, "menu")}>
            <p className="nav-label">{t(l, "workspace")}</p>
            <a
              className="rail-link"
              aria-current={route.page === "workspace" ? "page" : undefined}
              href="#/workspace"
            >
              <LayoutGrid />
              {t(l, "directory")}
            </a>
            <div className="pin-zone">
              <p className="nav-label">
                <Pin size={13} />
                {t(l, "pinned")}
              </p>
              {pinned.length ? (
                pinned.map((m) => (
                  <a
                    key={m.key}
                    className="rail-link"
                    href={moduleHref(m.key)}
                    aria-current={
                      route.page === "module" && m.key === route.key
                        ? "page"
                        : undefined
                    }
                  >
                    <BookOpen />
                    {m.title[l]}
                  </a>
                ))
              ) : (
                <p className="pin-hint">{t(l, "noPins")}</p>
              )}
            </div>
          </nav>
          <div className="rail-bottom">
            <a
              className="rail-link"
              href="#/settings"
              aria-current={route.page === "settings" ? "page" : undefined}
            >
              <Settings2 />
              {t(l, "prefs")}
            </a>
            <button className="rail-link" onClick={() => void signOut()}>
              <LogOut />
              {t(l, "signOut")}
            </button>
          </div>
        </div>
      </aside>
      </>}>{children}</AdminLayout>
  );
}
