import { useId, useLayoutEffect, useRef, useState } from "react";
import { ArrowUp, ArrowDown } from "lucide-react";
import type { ModuleDescriptor } from "@edupilot/api-client";
import { t, type CopyKey } from "../../app/copy";
import {
  orderedPinnedModules,
  movePinnedModule,
  useLocale,
  usePreferences,
  type Preferences,
} from "../../app/preferences";
import { Dialog, PageHeading } from "../../components/primitives";
type Name = Exclude<keyof Preferences, "pins" | "columns">;
const choices: {
  key: Name;
  label: CopyKey;
  help: CopyKey;
  values: readonly string[];
}[] = [
  {
    key: "locale",
    label: "language",
    help: "languageInfo",
    values: ["en", "fr"],
  },
  {
    key: "appearance",
    label: "appearance",
    help: "appearanceInfo",
    values: ["system", "light", "dark"],
  },
  {
    key: "accent",
    label: "accent",
    help: "accentInfo",
    values: ["pine", "indigo", "clay"],
  },
  {
    key: "density",
    label: "density",
    help: "densityInfo",
    values: ["comfortable", "compact"],
  },
  {
    key: "width",
    label: "width",
    help: "widthInfo",
    values: ["standard", "wide"],
  },
];
export function PreferencesPage({ modules, catalogStatus }: {
  modules: ModuleDescriptor[];
  catalogStatus: "ready" | "loading" | "unavailable";
}) {
  const l = useLocale(),
    pref = usePreferences((s) => s.value),
    update = usePreferences((s) => s.update),
    stored = usePreferences((s) => s.stored),
    id = useId(),
    [feedback, setFeedback] = useState<"saved" | "reset" | "resetFailed" | null>(null),
    [confirmReset, setConfirmReset] = useState(false),
    rows = useRef(new Map<string, HTMLLIElement>()),
    focusKey = useRef<string | null>(null);
  const pinned = orderedPinnedModules(modules, pref.pins);
  useLayoutEffect(() => {
    if (focusKey.current) {
      rows.current.get(focusKey.current)?.focus();
      focusKey.current = null;
    }
  }, [pref.pins]);
  const message = feedback === "resetFailed" ? t(l, "resetFailed")
    : !stored ? t(l, "storageUnavailable")
    : feedback === "reset" ? t(l, "resetDone")
    : feedback === "saved" ? t(l, "prefsSaved") : "";
  function move(key: string, direction: -1 | 1) {
    focusKey.current = key;
    update({ pins: movePinnedModule(pref.pins, modules, key, direction) });
    setFeedback("saved");
  }
  return (
    <>
      <PageHeading
        eyebrow={t(l, "personal")}
        title={t(l, "prefs")}
        description={t(l, "prefInfo")}
      />
      <section className="settings-sheet">
        {choices.map((c) => (
          <div key={c.key} className="setting-row">
            <div>
              <label htmlFor={id + c.key}>{t(l, c.label)}</label>
              <p id={id + c.help}>{t(l, c.help)}</p>
            </div>
            <select
              id={id + c.key}
              aria-describedby={id + c.help}
              value={pref[c.key]}
              onChange={(e) => {
                update({ [c.key]: e.target.value });
                setFeedback("saved");
              }}
            >
              {c.values.map((v) => (
                <option key={v} value={v}>
                  {v === "en"
                    ? "English"
                    : v === "fr"
                      ? "Français"
                      : t(l, v as CopyKey)}
                </option>
              ))}
            </select>
          </div>
        ))}
      </section>
      <section className="settings-sheet preference-section" aria-labelledby={id + "pins"}>
        <header className="preference-section-heading">
          <h2 id={id + "pins"}>{t(l, "pinnedNavigation")}</h2>
          <p className="lede" id={id + "pins-help"}>{t(l, "pinnedInfo")}</p>
        </header>
        {catalogStatus !== "ready" ? (
          <p className="preference-empty">{t(l, catalogStatus === "loading" ? "loading" : "pinsUnavailable")}</p>
        ) : pinned.length ? (
          <ol className="pinned-order" aria-labelledby={id + "pins"} aria-describedby={id + "pins-help"}>
            {pinned.map((module, index) => (
              <li key={module.key} aria-label={module.title[l]} tabIndex={-1}
                ref={element => { if (element) rows.current.set(module.key, element); else rows.current.delete(module.key); }}>
                <span>{module.title[l]}</span>
                <div className="row-actions">
                  <button type="button" className="btn" disabled={index === 0}
                    aria-label={t(l, "moveUp").replace("{module}", module.title[l])} onClick={() => move(module.key, -1)}>
                    <ArrowUp size={18} aria-hidden="true" />
                  </button>
                  <button type="button" className="btn" disabled={index === pinned.length - 1}
                    aria-label={t(l, "moveDown").replace("{module}", module.title[l])} onClick={() => move(module.key, 1)}>
                    <ArrowDown size={18} aria-hidden="true" />
                  </button>
                </div>
              </li>
            ))}
          </ol>
        ) : <div className="preference-empty"><p>{t(l, "noPins")}</p><a className="backlink" href="#/workspace">{t(l, "back")}</a></div>}
      </section>
      <section className="settings-sheet preference-section" aria-labelledby={id + "reset"}>
        <div className="setting-row">
          <div><h2 id={id + "reset"}>{t(l, "resetWorkspace")}</h2><p>{t(l, "resetInfo")}</p></div>
          <button type="button" className="btn danger" onClick={() => { setFeedback(null); setConfirmReset(true); }}>{t(l, "resetWorkspace")}</button>
        </div>
      </section>
      <p className="save-message" role="status">{confirmReset ? "" : message}</p>
      {confirmReset && <Dialog title={t(l, "resetQuestion")} description={t(l, "resetInfo")} onClose={() => { setConfirmReset(false); setFeedback(null); }}>
        <p className="save-message" role="alert">{feedback === "resetFailed" ? t(l, "resetFailed") : ""}</p>
        <div className="dialog-actions">
          <button type="button" className="btn" onClick={() => { setConfirmReset(false); setFeedback(null); }}>{t(l, "cancel")}</button>
          <button type="button" className="btn danger" onClick={() => {
            if (usePreferences.getState().reset()) { setConfirmReset(false); setFeedback("reset"); }
            else setFeedback("resetFailed");
          }}>{t(l, "resetConfirm")}</button>
        </div>
      </Dialog>}
    </>
  );
}
