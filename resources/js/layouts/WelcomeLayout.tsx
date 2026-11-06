import type { ReactNode } from "react";
import { Brand } from "../components/primitives";
import { t } from "../app/copy";
import { useLocale } from "../app/preferences";
export function WelcomeLayout({ children }: { children: ReactNode }) {
  const l = useLocale();
  return (
    <main className="login-page">
      <aside className="login-cover">
        <Brand />
        <div className="register-art" aria-hidden="true">
          <i />
          <i />
          <i />
          <i />
        </div>
        <div className="cover-caption">
          <span>{t(l, "brand")}</span>
          <h2>{t(l, "register")}</h2>
        </div>
      </aside>
      <section className="login-pane">{children}</section>
    </main>
  );
}
