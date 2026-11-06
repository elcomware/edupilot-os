import { WorkspaceTable } from "../../components/WorkspaceTable";
import { useState, type FormEvent } from "react";
import {
  useQuery,
  useMutation,
  useQueryClient,
  useQueries,
} from "@tanstack/react-query";
import { ChevronLeft, ChevronRight, GraduationCap, Plus } from "lucide-react";
import { ApiError, type Principal, type Placement } from "@edupilot/api-client";
import { api } from "../../app/session";
import { t } from "../../app/copy";
import { useLocale } from "../../app/preferences";
import { useUnsavedGuard } from "../../app/router";
import { errorKey } from "../../app/errors";
import { Dialog, StatePanel } from "../../components/primitives";
import { normalizeText } from "../collections/model";
import { sx } from "./copy";
export function validYear(value: string) {
  return (
    /^\d{4}-\d{4}$/.test(value) &&
    Number(value.slice(5)) === Number(value.slice(0, 4)) + 1
  );
}
const SIZE = 10;
const readable = (p: Principal) =>
  ["students.read", "workspace.read"].every((x) => p.permissions.includes(x));
const writable = (p: Principal) =>
  ["students.write", "workspace.write"].every((x) => p.permissions.includes(x));
function EnrollForm({
  principal,
  onClose,
  onSaved,
  onReview,
}: {
  principal: Principal;
  onClose: () => void;
  onSaved: () => void;
  onReview: () => void;
}) {
  const l = useLocale(),
    [offset, setOffset] = useState(0),
    [student, setStudent] = useState(""),
    [year, setYear] = useState(""),
    [grade, setGrade] = useState(""),
    [invalid, setInvalid] = useState<"year" | "grade" | null>(null),
    [discard, setDiscard] = useState(false);
  const roster = useQuery({
    queryKey: ["student-options", principal.school, principal.id, offset],
    queryFn: ({ signal }) => api.students({ limit: 25, offset }, signal),
    staleTime: 0,
  });
  const place = useMutation({
    mutationFn: () =>
      api.place({
        studentId: student,
        academicYear: year.trim(),
        grade: grade.trim(),
      }),
    onSuccess: onSaved,
  });
  const unknown =
    place.error instanceof ApiError && place.error.code === "write_unknown";
  useUnsavedGuard(Boolean(student || year || grade), t(l, "discardHint"));
  const cancel = () =>
    student || year || grade ? setDiscard(true) : onClose();
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (place.isPending || unknown) return;
    const issue = !validYear(year.trim())
      ? "year"
      : !normalizeText(grade, 48)
        ? "grade"
        : null;
    setInvalid(issue);
    if (issue)
      (e.currentTarget.elements.namedItem(issue) as HTMLInputElement)?.focus();
    if (!issue && student) place.mutate();
  }
  return (
    <Dialog
      title={discard ? t(l, "discard") : sx(l, "add")}
      busy={place.isPending}
      onClose={discard ? () => setDiscard(false) : cancel}
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
          <fieldset className="stack" disabled={place.isPending}>
            {roster.isError ? (
              <p role="alert" className="error-note">
                {t(l, errorKey(roster.error))}{" "}
                <button
                  type="button"
                  className="btn"
                  onClick={() => void roster.refetch()}
                >
                  {t(l, "retry")}
                </button>
              </p>
            ) : (
              <>
                <label>
                  {sx(l, "student")}
                  <select
                    required
                    value={student}
                    onChange={(e) => setStudent(e.target.value)}
                    disabled={roster.isPending}
                  >
                    <option value="">
                      {roster.isPending ? t(l, "loading") : sx(l, "choose")}
                    </option>
                    {roster.data?.items.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.givenName} {s.familyName} · {s.id}
                      </option>
                    ))}
                  </select>
                </label>
                {roster.data?.total === 0 && <p>{sx(l, "noActive")}</p>}
                {!!roster.data && roster.data.total > 25 && (
                  <nav
                    className="student-paging"
                    aria-label={sx(l, "studentPage")}
                  >
                    <button
                      className="btn"
                      type="button"
                      aria-label={sx(l, "prevStudents")}
                      disabled={!offset}
                      onClick={() => {
                        setOffset(offset - 25);
                        setStudent("");
                      }}
                    >
                      <ChevronLeft size={15} />
                    </button>
                    <span>
                      {offset + 1}–{offset + roster.data.items.length}{" "}
                      {t(l, "of")} {roster.data.total}
                    </span>
                    <button
                      className="btn"
                      type="button"
                      aria-label={sx(l, "nextStudents")}
                      disabled={offset + 25 >= roster.data.total}
                      onClick={() => {
                        setOffset(offset + 25);
                        setStudent("");
                      }}
                    >
                      <ChevronRight size={15} />
                    </button>
                  </nav>
                )}
              </>
            )}
            <label>
              {sx(l, "year")}
              <input
                name="year"
                required
                dir="ltr"
                value={year}
                aria-describedby={
                  invalid === "year"
                    ? "year-hint enrollment-error"
                    : "year-hint"
                }
                aria-invalid={invalid === "year" || undefined}
                onChange={(e) => {
                  setYear(e.target.value);
                  setInvalid(null);
                }}
              />
            </label>
            <small id="year-hint" className="field-help">
              {sx(l, "yearHint")}
            </small>
            <label>
              {sx(l, "grade")}
              <input
                name="grade"
                required
                aria-describedby={
                  invalid === "grade" ? "enrollment-error" : undefined
                }
                value={grade}
                aria-invalid={invalid === "grade" || undefined}
                onChange={(e) => {
                  setGrade(e.target.value);
                  setInvalid(null);
                }}
              />
            </label>
          </fieldset>
          {invalid && (
            <p id="enrollment-error" role="alert" className="error-note">
              {sx(l, invalid === "grade" ? "gradeRule" : "yearHint")}
            </p>
          )}
          {unknown && (
            <button className="btn" type="button" onClick={onReview}>
              {t(l, "checkList")}
            </button>
          )}
          {place.error && (
            <p role="alert" className="error-note">
              {place.error instanceof ApiError && place.error.status === 409
                ? sx(l, "enrollmentConflict")
                : t(l, errorKey(place.error))}
            </p>
          )}
          <footer className="dialog-actions">
            <button
              className="btn"
              type="button"
              onClick={cancel}
              disabled={place.isPending}
            >
              {t(l, "cancel")}
            </button>
            <button
              className="btn primary"
              disabled={
                place.isPending || !student || roster.isError || unknown
              }
            >
              {place.isPending ? t(l, "saving") : sx(l, "add")}
            </button>
          </footer>
        </form>
      )}
    </Dialog>
  );
}
function WithdrawForm({
  row,
  label,
  onClose,
  onSaved,
  onConflict,
}: {
  row: Placement;
  label: string;
  onClose: () => void;
  onSaved: () => void;
  onConflict: () => void;
}) {
  const l = useLocale(),
    mutation = useMutation({
      mutationFn: () => api.withdraw(row.id, row.version),
      onSuccess: onSaved,
      onError: (err) => {
        if (
          err instanceof ApiError &&
          (err.status === 409 || err.code === "write_unknown")
        )
          onConflict();
      },
    });
  const stale =
      mutation.error instanceof ApiError && mutation.error.status === 409,
    unknown =
      mutation.error instanceof ApiError &&
      mutation.error.code === "write_unknown";
  return (
    <Dialog
      title={sx(l, "withdraw")}
      onClose={onClose}
      busy={mutation.isPending}
    >
      <div className="stack">
        <p>
          <strong>{label}</strong> · {row.grade} · {row.academicYear}
        </p>
        <p>{sx(l, "withdrawHint")}</p>
        {mutation.error && (
          <p role="alert" className="error-note">
            {stale
              ? sx(l, "enrollmentConflict")
              : t(l, errorKey(mutation.error))}
          </p>
        )}
        <footer className="dialog-actions">
          <button
            className="btn"
            disabled={mutation.isPending}
            onClick={onClose}
          >
            {t(l, "cancel")}
          </button>
          <button
            className="btn danger"
            disabled={mutation.isPending || stale || unknown}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? t(l, "saving") : sx(l, "withdraw")}
          </button>
        </footer>
      </div>
    </Dialog>
  );
}
export function EnrollmentPanel({ principal }: { principal: Principal }) {
  const l = useLocale(),
    qc = useQueryClient(),
    [offset, setOffset] = useState(0),
    [add, setAdd] = useState(false),
    [withdraw, setWithdraw] = useState<Placement | null>(null),
    [message, setMessage] = useState<"placed" | "withdrawnNote" | null>(null);
  const scope = [principal.school, principal.id],
    prefix = ["placements", ...scope],
    canRead = readable(principal),
    canWrite = writable(principal);
  const list = useQuery({
    queryKey: [...prefix, offset],
    enabled: canRead,
    queryFn: ({ signal }) => api.placements({ limit: SIZE, offset }, signal),
  });
  const ids = [...new Set(list.data?.items.map((r) => r.studentId) ?? [])],
    names = useQueries({
      queries: ids.map((id) => ({
        queryKey: ["student-name", ...scope, id],
        queryFn: ({ signal }: { signal: AbortSignal }) =>
          api.student(id, signal),
      })),
    });
  const studentLabel = (id: string) => {
    const s = names[ids.indexOf(id)]?.data;
    return s ? s.givenName + " " + s.familyName : id;
  };
  function refresh() {
    void qc.invalidateQueries({ queryKey: prefix });
  }
  function saved(msg: "placed" | "withdrawnNote") {
    setMessage(msg);
    setAdd(false);
    setWithdraw(null);
    refresh();
  }
  if (!canRead) return null;
  return (
    <section className="collection-sheet special-sheet">
      <header className="sheet-heading">
        <GraduationCap size={18} />
        <h2>{sx(l, "placements")}</h2>
        {canWrite && (
          <button
            className="btn panel-action"
            onClick={() => {
              setMessage(null);
              setAdd(true);
            }}
          >
            <Plus size={15} />
            {sx(l, "add")}
          </button>
        )}
      </header>
      <p className="panel-lede">{sx(l, "placementHelp")}</p>
      {list.isPending ? (
        <StatePanel type="busy" title={t(l, "loading")} />
      ) : list.isError ? (
        <StatePanel
          type="error"
          title={t(l, "loadFailed")}
          body={t(l, errorKey(list.error))}
          action={
            <button className="btn" onClick={() => void list.refetch()}>
              {t(l, "retry")}
            </button>
          }
        />
      ) : !list.data?.items.length ? (
        <StatePanel type="empty" title={sx(l, "empty")} />
      ) : (
        <WorkspaceTable<Placement>
          view="enrollment:placements"
          label={sx(l, "placements")}
          rows={list.data.items}
          rowKey={(row) => row.id}
          columns={[
            {
              key: "student",
              label: sx(l, "student"),
              rowHeader: true,
              render: (row) => studentLabel(row.studentId),
            },
            {
              key: "year",
              label: sx(l, "year"),
              optional: true,
              render: (row) => row.academicYear,
            },
            {
              key: "grade",
              label: sx(l, "grade"),
              optional: true,
              render: (row) => row.grade,
            },
            {
              key: "status",
              label: t(l, "status"),
              optional: true,
              render: (row) =>
                row.status === "active" ? t(l, "active") : sx(l, "withdrawn"),
            },
            ...(canWrite
              ? [
                  {
                    key: "actions",
                    label: t(l, "actions"),
                    render: (row: Placement) =>
                      row.status === "active" && (
                        <button
                          className="btn"
                          onClick={() => {
                            setWithdraw(row);
                            setMessage(null);
                          }}
                        >
                          {sx(l, "withdraw")}
                        </button>
                      ),
                  },
                ]
              : []),
          ]}
        />
      )}
      {list.data && list.data.total > SIZE && (
        <nav className="pager" aria-label={sx(l, "placements")}>
          <span>
            {offset + 1}–{offset + list.data.items.length} {t(l, "of")}{" "}
            {list.data.total}
          </span>
          <div>
            <button
              className="btn"
              disabled={offset === 0}
              onClick={() => setOffset(offset - SIZE)}
            >
              {t(l, "previous")}
            </button>
            <button
              className="btn"
              disabled={offset + SIZE >= list.data.total}
              onClick={() => setOffset(offset + SIZE)}
            >
              {t(l, "next")}
            </button>
          </div>
        </nav>
      )}
      {message && (
        <p role="status" className="panel-feedback">
          {sx(l, message)}
        </p>
      )}
      {add && (
        <EnrollForm
          principal={principal}
          onClose={() => setAdd(false)}
          onSaved={() => saved("placed")}
          onReview={() => {
            setAdd(false);
            refresh();
          }}
        />
      )}
      {withdraw && (
        <WithdrawForm
          row={withdraw}
          label={studentLabel(withdraw.studentId)}
          onClose={() => setWithdraw(null)}
          onSaved={() => saved("withdrawnNote")}
          onConflict={refresh}
        />
      )}
    </section>
  );
}
