import { useEffect, useId, useRef, type ReactNode } from "react";
import {
  BookOpen,
  LoaderCircle,
  AlertCircle,
  FolderOpen,
  X,
} from "lucide-react";
import { useLocale } from "../app/preferences";
import { t } from "../app/copy";
export function Brand({ wordmark = true }: { wordmark?: boolean }) {
  return (
    <span className="brand" aria-label="EduPilot OS">
      <svg aria-hidden="true" viewBox="0 0 38 38" width="36" height="36">
        <path d="M4 10.5h9l3-5h17v24H19l-4 5H4z" fill="currentColor" />
        <path
          d="M10 16h16m-16 6h11"
          fill="none"
          stroke="var(--mark-cut)"
          strokeWidth="2.4"
        />
      </svg>
      {wordmark && (
        <span>
          EduPilot <b>OS</b>
        </span>
      )}
    </span>
  );
}
export function PageHeading({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow: string;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <header className="page-heading">
      <div>
        <p className="eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        {description && <p className="lede">{description}</p>}
      </div>
      {action}
    </header>
  );
}
export function StatePanel({
  type,
  title,
  body,
  action,
}: {
  type: "busy" | "empty" | "error";
  title: string;
  body?: string;
  action?: ReactNode;
}) {
  const Icon =
    type === "busy"
      ? LoaderCircle
      : type === "error"
        ? AlertCircle
        : FolderOpen;
  return (
    <section
      className="state-panel"
      aria-live="polite"
      aria-busy={type === "busy"}
    >
      <Icon
        className={type === "busy" ? "spin" : ""}
        aria-hidden="true"
        size={24}
      />
      <h2>{title}</h2>
      {body && <p>{body}</p>}
      {action}
    </section>
  );
}
export function Dialog({
  title,
  children,
  onClose,
  busy = false,
  description,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  busy?: boolean;
  description?: string;
}) {
  const ref = useRef<HTMLDialogElement>(null),
    id = useId(),
    locale = useLocale();
  useEffect(() => {
    const d = ref.current,
      trigger = document.activeElement as HTMLElement | null;
    d?.showModal();
    return () => {
      d?.close();
      if (trigger?.isConnected) trigger.focus();
      else document.querySelector<HTMLElement>("#content")?.focus();
    };
  }, []);
  return (
    <dialog
      className="dialog"
      aria-labelledby={id}
      ref={ref}
      onKeyDown={(event) => {
        if (
          event.key !== "Tab" || event.defaultPrevented ||
          event.altKey || event.ctrlKey || event.metaKey
        ) return;
        // showModal makes the page inert, but boundary Tab can still move to
        // browser chrome. Wrap only the endpoints; leave middle controls alone.
        const dialog = event.currentTarget;
        const controls = Array.from(dialog.querySelectorAll<HTMLElement>(
          "a[href], button, input, select, textarea, summary, [tabindex]",
        )).filter((control) => {
          if (
            control.tabIndex < 0 || control.matches(":disabled, input[type='hidden']") ||
            control.closest("[hidden], [inert]")
          ) return false;
          const visibility = getComputedStyle(control).visibility;
          if (visibility === "hidden" || visibility === "collapse") return false;
          for (let node: HTMLElement | null = control; node && node !== dialog; node = node.parentElement) {
            if (getComputedStyle(node).display === "none") return false;
            if (node instanceof HTMLDetailsElement && !node.open) {
              const summary = node.querySelector(":scope > summary");
              if (!summary?.contains(control)) return false;
            }
          }
          return true;
        });
        const first = controls[0], last = controls[controls.length - 1];
        if (!first) {
          event.preventDefault();
        } else if (event.shiftKey && document.activeElement === first) {
          event.preventDefault();
          last.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first.focus();
        }
      }}
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
    >
      <div className="dialog-head">
        <div>
          <BookOpen aria-hidden="true" size={20} />
          <h2 id={id}>{title}</h2>
        </div>
        <button
          type="button"
          className="icon-button"
          aria-label={t(locale, "cancel")}
          disabled={busy}
          onClick={onClose}
        >
          <X size={19} />
        </button>
      </div>
      {description && <p>{description}</p>}
      {children}
    </dialog>
  );
}
