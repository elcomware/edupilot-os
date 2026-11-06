import { useMemo, useState } from "react";
import type { ModuleDescriptor } from "@edupilot/api-client";
import {
  ArrowUpRight,
  BookOpen,
  Building2,
  GraduationCap,
  Layers3,
  MessagesSquare,
  Pin,
  Search,
  Wallet,
  Waypoints,
} from "lucide-react";
import { t, groupLabel } from "@edupilot/core/web/app/copy";
import { useLocale, usePreferences } from "@edupilot/core/web/app/preferences";
import { moduleHref } from "@edupilot/core/web/app/router";
import { PageHeading, StatePanel } from "@edupilot/core/web/components/primitives";
const icons = {
  platform: Layers3,
  lifecycle: Waypoints,
  academics: GraduationCap,
  finance: Wallet,
  operations: Building2,
  engagement: MessagesSquare,
};
function fold(s: string, l: string) {
  return s
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLocaleLowerCase(l);
}
export function Directory({ modules, icons: moduleIcons }: { modules: ModuleDescriptor[]; icons: ReadonlyMap<string, string> }) {
  const l = useLocale(),
    pin = usePreferences((s) => s.pin),
    pins = usePreferences((s) => s.value.pins),
    [term, setTerm] = useState(""),
    [group, setGroup] = useState("");
  const groups = useMemo(
    () => Array.from(new Set(modules.map((m) => m.group))),
    [modules],
  );
  const visible = modules.filter(
    (m) =>
      (!group || m.group === group) &&
      fold(m.title[l] + " " + m.key, l).includes(fold(term.trim(), l)),
  );
  return (
    <>
      <PageHeading
        eyebrow={t(l, "workspace")}
        title={t(l, "directory")}
        description={t(l, "directoryInfo")}
      />
      <div className="directory-toolbar">
        <div className="search-field">
          <Search size={18} />
          <input
            type="search"
            aria-label={t(l, "search")}
            placeholder={t(l, "search")}
            value={term}
            onChange={(e) => setTerm(e.target.value)}
          />
        </div>
        <select
          aria-label={t(l, "group")}
          value={group}
          onChange={(e) => setGroup(e.target.value)}
        >
          <option value="">{t(l, "allGroups")}</option>
          {groups.map((g) => (
            <option key={g} value={g}>
              {groupLabel(l, g)}
            </option>
          ))}
        </select>
      </div>
      {!modules.length ? (
        <StatePanel
          type="empty"
          title={t(l, "emptyModules")}
          body={t(l, "emptyModulesHint")}
        />
      ) : !visible.length ? (
        <StatePanel
          type="empty"
          title={t(l, "noMatches")}
          action={
            <button
              className="btn"
              onClick={() => {
                setTerm("");
                setGroup("");
              }}
            >
              {t(l, "clearFilters")}
            </button>
          }
        />
      ) : (
        groups.map((g) => {
          const rows = visible.filter((m) => m.group === g);
          if (!rows.length) return null;
          const Icon = icons[g as keyof typeof icons] || BookOpen;
          return (
            <section key={g} className="index-group">
              <header>
                <Icon aria-hidden="true" size={18} />
                <h2>{groupLabel(l, g)}</h2>
                <span className="count">
                  {new Intl.NumberFormat(l).format(rows.length)}
                </span>
              </header>
              <ul className="module-grid">
                {rows.map((m) => {
                  const pinned = pins.includes(m.key);
                  return (
                    <li key={m.key} className="module-tab">
                      <a href={moduleHref(m.key)}>
                        {moduleIcons.has(m.key) && <img className="module-symbol" src={moduleIcons.get(m.key)} width="32" height="32" alt="" />}
                        <span>{m.title[l]}</span>
                        <ArrowUpRight size={17} aria-hidden="true" />
                      </a>
                      <button
                        className={"pin-button" + (pinned ? " pinned" : "")}
                        aria-pressed={pinned}
                        aria-label={
                          t(l, pinned ? "unpin" : "pin") + " " + m.title[l]
                        }
                        onClick={() => pin(m.key)}
                      >
                        <Pin size={15} />
                      </button>
                    </li>
                  );
                })}
              </ul>
            </section>
          );
        })
      )}
    </>
  );
}
