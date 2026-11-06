import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { ArrowRight, KeyRound, Languages } from "lucide-react";
import { api, useAuth } from "@edupilot/core/web/app/session";
import { t, type CopyKey } from "@edupilot/core/web/app/copy";
import { errorKey } from "@edupilot/core/web/app/errors";
import { useLocale, usePreferences } from "@edupilot/core/web/app/preferences";
import { WelcomeLayout } from "@edupilot/core/web/layouts/WelcomeLayout";
export function Login() {
  const l = useLocale(),
    update = usePreferences((s) => s.update),
    start = useAuth((s) => s.start),
    expired = useAuth((s) => s.expired);
  const [school, setSchool] = useState(""),
    [email, setEmail] = useState(""),
    [password, setPassword] = useState("");
  const [invalid, setInvalid] = useState<
      "school" | "email" | "password" | null
    >(null),
    revocationUnknown = useAuth((s) => s.revocationUnknown);
  const fieldErrors: Record<string, CopyKey> = {
    school: "schoolRequired",
    email: "emailRequired",
    password: "passwordRequired",
  };
  const login = useMutation({
    mutationFn: api.login,
    onSuccess: start,
    onError: () => setPassword(""),
  });
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (login.isPending) return;
    const emailField = e.currentTarget.elements.namedItem(
      "email",
    ) as HTMLInputElement;
    const issue = !school.trim()
      ? "school"
      : !email.trim() || !emailField.validity.valid
        ? "email"
        : !password
          ? "password"
          : null;
    setInvalid(issue);
    if (issue) {
      (e.currentTarget.elements.namedItem(issue) as HTMLInputElement).focus();
      return;
    }
    login.mutate({ school: school.trim(), email: email.trim(), password });
  }
  return (
    <WelcomeLayout>
        <div className="login-locale">
          <Languages aria-hidden="true" size={16} />
          <select
            aria-label={t(l, "language")}
            value={l}
            onChange={(e) =>
              update({ locale: e.target.value === "fr" ? "fr" : "en" })
            }
          >
            <option value="en">English</option>
            <option value="fr">Français</option>
          </select>
        </div>
        <div className="login-box">
          <p className="eyebrow">
            <KeyRound size={16} aria-hidden="true" />
            {t(l, "access")}
          </p>
          <h1>{t(l, "welcome")}</h1>
          <p className="lede">{t(l, "loginInfo")}</p>
          {expired && (
            <p role="status" className="notice">
              {t(l, "sessionExpired")}
            </p>
          )}
          {revocationUnknown && (
            <p role="status" className="notice">
              {t(l, "logoutUnconfirmed")}
            </p>
          )}
          <form noValidate onSubmit={submit} className="stack">
            <label>
              {t(l, "school")}
              <input
                name="school"
                aria-invalid={invalid === "school" || undefined}
                autoComplete="organization"
                required
                value={school}
                onChange={(e) => {
                  setSchool(e.target.value);
                  if (invalid === "school") setInvalid(null);
                }}
                aria-describedby={
                  invalid === "school"
                    ? "school-help login-error"
                    : "school-help"
                }
              />
            </label>
            <small id="school-help" className="field-help">
              {t(l, "schoolHint")}
            </small>
            <label>
              {t(l, "email")}
              <input
                name="email"
                aria-invalid={invalid === "email" || undefined}
                aria-describedby={
                  invalid === "email" ? "login-error" : undefined
                }
                required
                autoComplete="username"
                type="email"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  if (invalid === "email") setInvalid(null);
                }}
              />
            </label>
            <label>
              {t(l, "password")}
              <input
                name="password"
                aria-invalid={invalid === "password" || undefined}
                aria-describedby={
                  invalid === "password" ? "login-error" : undefined
                }
                required
                autoComplete="current-password"
                type="password"
                value={password}
                onChange={(e) => {
                  setPassword(e.target.value);
                  if (invalid === "password") setInvalid(null);
                }}
              />
            </label>
            {invalid && (
              <p role="alert" id="login-error" className="error-note">
                {t(l, fieldErrors[invalid])}
              </p>
            )}
            {login.error && !invalid && (
              <p className="error-note" role="alert">
                {t(l, errorKey(login.error, true))}
              </p>
            )}
            <button
              className="btn primary login-submit"
              disabled={login.isPending}
            >
              {t(l, login.isPending ? "signingIn" : "signIn")}
              <ArrowRight size={17} />
            </button>
          </form>
        </div>
    </WelcomeLayout>
  );
}
