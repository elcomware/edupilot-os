import { useEffect, useState, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronLeft, ChevronRight, Plus } from "lucide-react";
import type { Page, Principal } from "@edupilot/api-client";
import { api } from "../../app/session";
import { t, groupLabel, type CopyKey } from "../../app/copy";
import { useLocale } from "../../app/preferences";
import { errorKey } from "../../app/errors";
import { StatePanel, PageHeading } from "../../components/primitives";
import { ModulePanel } from "../specialized/ModulePanel";
import { CollectionTable } from "./CollectionTable";
import { RecordDialog, type Editor } from "./RecordDialog";
import { PAGE_SIZE, collectionAccess, type RegisterRow } from "./model";
export function WorkspacePage({
  moduleKey,
  principal,
}: {
  moduleKey: string;
  principal: Principal;
}) {
  const l = useLocale(),
    qc = useQueryClient(),
    [offset, setOffset] = useState(0),
    [editor, setEditor] = useState<Editor | null>(null),
    [feedback, setFeedback] = useState<CopyKey | null>(null);
  const registerHeading = useRef<HTMLHeadingElement>(null);
  const scope = [principal.school, principal.id],
    info = useQuery({
      queryKey: ["descriptor", ...scope, moduleKey],
      queryFn: ({ signal }) => api.module(moduleKey, signal),
    });
  const descriptor = info.data,
    access = descriptor ? collectionAccess(descriptor, principal) : null,
    students = moduleKey === "students";
  const collectionKey = ["register", ...scope, moduleKey];
  const list = useQuery<Page<RegisterRow>>({
    queryKey: [...collectionKey, offset],
    enabled: !!access?.list,
    queryFn: ({ signal }) =>
      students
        ? api.students({ offset, limit: PAGE_SIZE }, signal)
        : api.workspace(moduleKey, { offset, limit: PAGE_SIZE }, signal),
  });
  const data = list.data,
    total = data?.total ?? 0;
  useEffect(() => {
    if (data && offset > 0 && offset >= total)
      setOffset(Math.max(0, (Math.ceil(total / PAGE_SIZE) - 1) * PAGE_SIZE));
  }, [data, offset, total]);
  const back = (
    <a className="backlink" href="#/workspace">
      <ChevronLeft size={15} />
      {t(l, "back")}
    </a>
  );
  if (info.isPending)
    return (
      <>
        {back}
        <StatePanel type="busy" title={t(l, "loading")} />
      </>
    );
  if (info.isError || !descriptor)
    return (
      <>
        {back}
        <StatePanel
          type="error"
          title={t(l, "unavailable")}
          body={t(l, errorKey(info.error))}
          action={
            <button className="btn" onClick={() => void info.refetch()}>
              {t(l, "retry")}
            </button>
          }
        />
      </>
    );
  if (!access?.list)
    return (
      <>
        {back}
        <PageHeading
          eyebrow={groupLabel(l, descriptor.group)}
          title={descriptor.title[l]}
        />
        <ModulePanel moduleKey={moduleKey} principal={principal} />
        <StatePanel type="error" title={t(l, "denied")} />
      </>
    );
  const add = () => {
    setFeedback(null);
    setEditor({ kind: "create" });
  };
  async function latest(id: string) {
    if (students) return api.student(id);
    return api.entry(moduleKey, id);
  }
  function onSaved(msg: CopyKey) {
    setEditor(null);
    setFeedback(msg);
    void qc.invalidateQueries({ queryKey: collectionKey });
    if (msg === "studentArchived")
      requestAnimationFrame(() => registerHeading.current?.focus());
  }
  const number = (n: number) => new Intl.NumberFormat(l).format(n);
  return (
    <>
      {back}
      <PageHeading
        eyebrow={groupLabel(l, descriptor.group)}
        title={descriptor.title[l]}
        description={t(l, students ? "studentInfo" : "entryInfo")}
        action={
          access.create && (
            <button className="btn primary" onClick={add}>
              <Plus size={16} />
              {t(l, students ? "addStudent" : "addEntry")}
            </button>
          )
        }
      />
      <ModulePanel moduleKey={moduleKey} principal={principal} />
      <section className="collection-sheet">
        <div className="sheet-heading">
          <h2 ref={registerHeading} tabIndex={-1}>
            {t(l, students ? "studentRegister" : "entries")}
          </h2>
          {data && <span className="count">{number(total)}</span>}
        </div>
        <div className="sr-only" role="status">
          {list.isFetching ? t(l, "loading") : ""}
        </div>
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
        ) : data?.items.length ? (
          <CollectionTable
            moduleKey={moduleKey}
            rows={data.items}
            students={students}
            canEdit={access.edit}
            canArchive={access.archive}
            onEdit={(row) => {
              setFeedback(null);
              setEditor({ kind: "edit", row });
            }}
            onArchive={(row) => {
              setFeedback(null);
              setEditor({ kind: "archive", row });
            }}
          />
        ) : (
          <StatePanel
            type="empty"
            title={t(l, students ? "noStudents" : "noEntries")}
            body={
              access.create
                ? t(l, students ? "addFirstStudent" : "addFirstEntry")
                : undefined
            }
          />
        )}
        {data && total > 0 && (
          <nav className="pager" aria-label={t(l, "records")}>
            <span>
              {number(Math.min(offset + 1, total))}–
              {number(Math.min(offset + data.items.length, total))} {t(l, "of")}{" "}
              {number(total)}
            </span>
            <div>
              <button
                className="btn"
                aria-label={t(l, "previous")}
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
              >
                <ChevronLeft size={17} />
              </button>
              <button
                className="btn"
                aria-label={t(l, "next")}
                disabled={offset + PAGE_SIZE >= total}
                onClick={() => setOffset(offset + PAGE_SIZE)}
              >
                <ChevronRight size={17} />
              </button>
            </div>
          </nav>
        )}
      </section>
      <p role="status" className="save-message">
        {feedback ? t(l, feedback) : ""}
      </p>
      {editor && (
        <RecordDialog
          module={moduleKey}
          editor={editor}
          onSaved={onSaved}
          onClose={() => setEditor(null)}
          latest={latest}
          onReviewList={() =>
            void qc.invalidateQueries({ queryKey: collectionKey })
          }
        />
      )}
    </>
  );
}
