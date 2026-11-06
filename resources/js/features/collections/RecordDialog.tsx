import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { api } from "../../app/session";
import { ApiError } from "@edupilot/api-client";
import { useLocale } from "../../app/preferences";
import { t, type CopyKey } from "../../app/copy";
import { useUnsavedGuard } from "../../app/router";
import { errorKey } from "../../app/errors";
import { Dialog } from "../../components/primitives";
import { normalizeText, type RegisterRow } from "./model";
export type Editor =
  | { kind: "create" }
  | { kind: "edit"; row: RegisterRow };
type Props = {
  module: string;
  editor: Editor;
  onClose: () => void;
  onSaved: (msg: CopyKey) => void;
  latest: (id: string) => Promise<RegisterRow | undefined>;
  onReviewList?: () => void;
};
function fields(row?: RegisterRow) {
  return {
    title: row?.title ?? "",
    status: row?.status ?? "open",
  };
}
export function RecordDialog({
  module,
  editor,
  onClose,
  onSaved,
  latest,
  onReviewList,
}: Props) {
  const l = useLocale(),
    creating = editor.kind === "create";
  const [base, setBase] = useState("row" in editor ? editor.row : undefined),
    [values, setValues] = useState(() => fields(base)),
    [discard, setDiscard] = useState(false),
    [invalid, setInvalid] = useState<string | null>(null),
    [missing, setMissing] = useState(false);
  const dirty =
    JSON.stringify(values) !== JSON.stringify(fields(base));
  useUnsavedGuard(dirty, t(l, "discardHint"));
  const save = useMutation({
    mutationFn: async () => {
      if (base)
        await api.updateEntry(module, base.id, {
          title: values.title.trim(),
          version: base.version,
          status: values.status,
        });
      else await api.createEntry(module, { title: values.title.trim() });
    },
    onSuccess: () => onSaved("entrySaved"),
  });
  const refresh = useMutation({
    mutationFn: () => latest(base!.id),
    onSuccess: (row) => {
      if (!row) {
        setMissing(true);
        return;
      }
      setBase(row);
      setValues(fields(row));
      setInvalid(null);
      save.reset();
    },
  });
  const conflict =
    save.error instanceof ApiError && [409, 412].includes(save.error.status);
  const unknown =
    save.error instanceof ApiError && save.error.code === "write_unknown";
  const busy = save.isPending || refresh.isPending,
    requestClose = () => (dirty ? setDiscard(true) : onClose());
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (busy || missing || conflict || unknown) return;
    const key = !normalizeText(values.title,160) ? "title"
        : null;
    if (key) {
      setInvalid(key);
      (e.currentTarget.elements.namedItem(key) as HTMLInputElement)?.focus();
      return;
    }
    setInvalid(null);
    save.mutate();
  }
  const title=t(l, creating ? "addEntry":"editEntry");
  const field = (name: "title") => (
    <label>
      {t(l, name)}
      <input
        name={name}
        required
        value={values[name]}
        aria-invalid={invalid === name || undefined}
        aria-describedby={invalid === name ? "validation-message" : undefined}
        onChange={(e) => {
          setValues({ ...values, [name]: e.target.value });
          setInvalid(null);
        }}
      />
    </label>
  );
  return (
    <Dialog
      title={discard ? t(l, "discard") : title}
      onClose={discard ? () => setDiscard(false) : requestClose}
      busy={busy}
    >
      {discard ? (
        <div className="stack">
          <p>{t(l, "discardHint")}</p>
          <footer className="dialog-actions">
            <button className="btn" onClick={() => setDiscard(false)}>
              {t(l, "keepEditing")}
            </button>
            <button className="btn danger" onClick={onClose}>
              {t(l, "discardAction")}
            </button>
          </footer>
        </div>
      ) : (
        <form noValidate className="stack" onSubmit={submit}>
          <fieldset disabled={busy} className="stack">
            {field("title")}
                {!creating && (
                  <label>
                    {t(l, "status")}
                    <select
                      value={values.status}
                      onChange={(e) =>
                        setValues({ ...values, status: e.target.value })
                      }
                    >
                      <option value="open">{t(l, "open")}</option>
                      <option value="closed">{t(l, "closed")}</option>
                    </select>
                  </label>
                )}
          </fieldset>
          {invalid && (
            <p id="validation-message" className="error-note" role="alert">
              {t(l, "titleRule")}
            </p>
          )}
          {missing ? (
            <p role="alert" className="error-note">
              {t(l, "missing")}
            </p>
          ) : (
            save.error && (
              <p role="alert" className="error-note">
                {t(l, errorKey(save.error))}
              </p>
            )
          )}
          {refresh.error && (
            <p role="alert" className="error-note">
              {t(l, errorKey(refresh.error))}
            </p>
          )}
          {unknown && (
            <button
              type="button"
              className="btn"
              onClick={() => {
                onReviewList?.();
                onClose();
              }}
            >
              {t(l, "checkList")}
            </button>
          )}
          {conflict && !missing && (
            <button
              type="button"
              className="btn"
              onClick={() => refresh.mutate()}
              disabled={busy}
            >
              {t(l, "loadLatest")}
            </button>
          )}
          <footer className="dialog-actions">
            <button
              type="button"
              className="btn"
              onClick={requestClose}
              disabled={busy}
            >
              {t(l, "cancel")}
            </button>
            <button
              className="btn primary"
              disabled={busy || missing || conflict || unknown}
            >
              {t(
                l,
                busy ? "saving" : creating ? "saveEntry" : "saveChanges",
              )}
            </button>
          </footer>
        </form>
      )}
    </Dialog>
  );
}
