import { WorkspaceTable, type Column } from "../../components/WorkspaceTable";
import { Pencil } from "lucide-react";
import { t } from "../../app/copy";
import { useLocale } from "../../app/preferences";
import { rowName, type RegisterRow } from "./model";
type Props = {
  moduleKey: string;
  rows: RegisterRow[];
  canEdit: boolean;
  onEdit: (r: RegisterRow) => void;
};
export function CollectionTable({ moduleKey, rows, canEdit, onEdit }: Props) {
  const l = useLocale();
  const columns: Column<RegisterRow>[] = [
    {
      key: "name",
      label: t(l, "title"),
      rowHeader: true,
      render: rowName,
    },
    {
      key: "status",
      label: t(l, "status"),
      optional: true,
      render: (row) => {
        const status = row.status;
        const known = status === "open" || status === "closed";
        return (
          <span className={"badge " + (status === "open" ? "active" : "")}>
            <span aria-hidden="true" />
            {known ? t(l, status) : status}
          </span>
        );
      },
    },
  ];
  if (canEdit)
    columns.push({
      key: "actions",
      label: t(l, "actions"),
      className: "action-col",
      render: (row) => {
        const label = rowName(row);
        return (
          <div className="row-actions">
            <button
              className="icon-button"
              aria-label={t(l, "edit") + " " + label}
              title={t(l, "edit")}
              onClick={() => onEdit(row)}
            >
              <Pencil size={16} />
            </button>
          </div>
        );
      },
    });
  return (
    <WorkspaceTable
      view={"collection:" + moduleKey}
      label={t(l, "entries")}
      rows={rows}
      rowKey={(row) => row.id}
      columns={columns}
    />
  );
}
