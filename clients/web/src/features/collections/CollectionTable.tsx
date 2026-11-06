import { WorkspaceTable, type Column } from "../../components/WorkspaceTable";
import { Archive, Pencil } from "lucide-react";
import { t } from "../../app/copy";
import { useLocale } from "../../app/preferences";
import { rowName, type RegisterRow } from "./model";
type Props = {
  moduleKey: string;
  rows: RegisterRow[];
  students: boolean;
  canEdit: boolean;
  canArchive: boolean;
  onEdit: (r: RegisterRow) => void;
  onArchive: (r: RegisterRow) => void;
};
export function CollectionTable({
  moduleKey,
  rows,
  students,
  canEdit,
  canArchive,
  onEdit,
  onArchive,
}: Props) {
  const l = useLocale();
  const columns: Column<RegisterRow>[] = [
    {
      key: "name",
      label: t(l, students ? "name" : "title"),
      rowHeader: true,
      render: rowName,
    },
    {
      key: "status",
      label: t(l, "status"),
      optional: true,
      render: (row) => {
        const status = row.status;
        const known =
          status === "open" ||
          status === "closed" ||
          status === "active" ||
          status === "archived";
        return (
          <span
            className={
              "badge " +
              (status === "active" || status === "open" ? "active" : "")
            }
          >
            <span aria-hidden="true" />
            {known ? t(l, status) : status}
          </span>
        );
      },
    },
  ];
  if (canEdit || canArchive)
    columns.push({
      key: "actions",
      label: t(l, "actions"),
      className: "action-col",
      render: (row) => {
        if (row.status === "archived") return null;
        const label = rowName(row);
        return (
          <div className="row-actions">
            {canEdit && (
              <button
                className="icon-button"
                aria-label={t(l, "edit") + " " + label}
                title={t(l, "edit")}
                onClick={() => onEdit(row)}
              >
                <Pencil size={16} />
              </button>
            )}
            {canArchive && (
              <button
                className="icon-button"
                aria-label={t(l, "archive") + " " + label}
                title={t(l, "archive")}
                onClick={() => onArchive(row)}
              >
                <Archive size={16} />
              </button>
            )}
          </div>
        );
      },
    });
  return (
    <WorkspaceTable
      view={"collection:" + moduleKey}
      label={t(l, students ? "studentRegister" : "entries")}
      rows={rows}
      rowKey={(row) => row.id}
      columns={columns}
    />
  );
}
