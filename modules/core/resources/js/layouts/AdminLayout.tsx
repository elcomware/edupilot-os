import type { ReactNode, Ref } from "react";
import { SquareLibrary, ChevronRight } from "lucide-react";
export function AdminLayout({
  navigation,
  scope,
  title,
  mainRef,
  children,
}: {
  navigation: ReactNode;
  scope: string;
  title: string;
  mainRef?: Ref<HTMLElement>;
  children: ReactNode;
}) {
  return (
    <div className="app-shell">
      {navigation}
      <div className="workplane">
        <header className="topline">
          <span className="school-label">
            <SquareLibrary size={18} aria-hidden="true" />
            <span>{scope}</span>
          </span>
          <span className="crumb">
            <ChevronRight size={15} aria-hidden="true" />
            {title}
          </span>
        </header>
        <main id="content" ref={mainRef} tabIndex={-1} className="main-area">
          {children}
        </main>
      </div>
    </div>
  );
}
