import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { api } from "../../app/session";
import { ApiError } from "@edupilot/api-client";
import { useLocale } from "../../app/preferences";
import { t, type CopyKey } from "../../app/copy";
import { useUnsavedGuard } from "../../app/router";
import { errorKey } from "../../app/errors";
import { Dialog } from "../../components/primitives";
import { isStudent, normalizeText, rowName, type RegisterRow } from "./model";
export type Editor =
  | { kind: "create" }
  | { kind: "edit" | "archive"; row: RegisterRow };
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
    given: row && isStudent(row) ? row.givenName : "",
    family: row && isStudent(row) ? row.familyName : "",
    title: row && !isStudent(row) ? row.title : "",
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
    students = module === "students",
    archive = editor.kind === "archive",
    creating = editor.kind === "create";
  const [base, setBase] = useState("row" in editor ? editor.row : undefined),
    [values, setValues] = useState(() => fields(base)),
    [discard, setDiscard] = useState(false),
    [invalid, setInvalid] = useState<string | null>(null),
    [missing, setMissing] = useState(false);
  const dirty =
    !archive && JSON.stringify(values) !== JSON.stringify(fields(base));
  useUnsavedGuard(dirty, t(l, "discardHint"));
  const save = useMutation({
    mutationFn: async () => {
      if (archive && base) {
        await api.archiveStudent(base.id, base.version);
        return;
      }
      if (students) {
        const names = {
          givenName: values.given.trim(),
          familyName: values.family.trim(),
        };
        if (base)
          await api.updateStudent(base.id, { ...names, version: base.version });
        else await api.createStudent(names);
      } else if (base)
        await api.updateEntry(module, base.id, {
          title: values.title.trim(),
          version: base.version,
          status: values.status,
        });
      else await api.createEntry(module, { title: values.title.trim() });
    },
    onSuccess: () =>
      onSaved(
        archive ? "studentArchived" : students ? "studentSaved" : "entrySaved",
      ),
  });
  const refresh = useMutation({
    mutationFn: () => latest(base!.id),
    onSuccess: (row) => {
      if (!row || row.status === "archived") {
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
    const key = students
      ? !normalizeText(values.given, 100)
        ? "given"
        : !normalizeText(values.family, 100)
          ? "family"
          : null
      : !normalizeText(values.title, 160)
        ? "title"
        : null;
    if (!archive && key) {
      setInvalid(key);
      (e.currentTarget.elements.namedItem(key) as HTMLInputElement)?.focus();
      return;
    }
    setInvalid(null);
    save.mutate();
  }
  const title = t(
    l,
    archive
      ? "archiveStudent"
      : students
        ? creating
          ? "addStudent"
          : "editStudent"
        : creating
          ? "addEntry"
          : "editEntry",
  );
  const field = (name: "given" | "family" | "title") => (
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
            {archive && base ? (
              <>
                <p className="archive-name">{rowName(base)}</p>
                <p>{t(l, "archiveNote")}</p>
              </>
            ) : students ? (
              <>
                {field("given")}
                {field("family")}
              </>
            ) : (
              <>
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
              </>
            )}
          </fieldset>
          {invalid && (
            <p id="validation-message" className="error-note" role="alert">
              {t(l, students ? "nameRule" : "titleRule")}
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
              className={"btn " + (archive ? "danger" : "primary")}
              disabled={busy || missing || conflict || unknown}
            >
              {t(
                l,
                busy
                  ? "saving"
                  : archive
                    ? "archiveStudent"
                    : creating
                      ? students
                        ? "saveStudent"
                        : "saveEntry"
                      : "saveChanges",
              )}
            </button>
          </footer>
        </form>
      )}
    </Dialog>
  );
}
