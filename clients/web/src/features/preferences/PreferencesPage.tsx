import { useId, useState } from "react";
import { t, type CopyKey } from "../../app/copy";
import {
  useLocale,
  usePreferences,
  type Preferences,
} from "../../app/preferences";
import { PageHeading } from "../../components/primitives";
type Name = Exclude<keyof Preferences, "pins">;
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
export function PreferencesPage() {
  const l = useLocale(),
    pref = usePreferences((s) => s.value),
    update = usePreferences((s) => s.update),
    stored = usePreferences((s) => s.stored),
    id = useId(),
    [changed, setChanged] = useState(false);
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
                setChanged(true);
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
      <p className="save-message" role="status">
        {!stored
          ? t(l, "storageUnavailable")
          : changed
            ? t(l, "prefsSaved")
            : ""}
      </p>
    </>
  );
}
