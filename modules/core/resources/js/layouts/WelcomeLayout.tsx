import mark from "@edupilot/core/brand/svg/edupilot-mark-dark.svg";
import type { ReactNode } from "react";
import { Brand } from "@edupilot/core/web/components/primitives";
import { t } from "@edupilot/core/web/app/copy";
import { useLocale } from "@edupilot/core/web/app/preferences";
export function WelcomeLayout({ children }: { children: ReactNode }) {
  const l = useLocale();
  return (
    <main className="login-page">
      <aside className="login-cover">
        <Brand />
        <img className="cover-monogram" src={mark} alt="" />
        <div className="cover-caption">
          <span>{t(l, "brand")}</span>
          <h2>{t(l, "register")}</h2>
        </div>
      </aside>
      <section className="login-pane">{children}</section>
    </main>
  );
}
