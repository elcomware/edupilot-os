import { Archive, Pencil } from "lucide-react";
import { t } from "../../app/copy";
import { useLocale } from "../../app/preferences";
import { rowName, type RegisterRow } from "./model";
type Props = {
  rows: RegisterRow[];
  students: boolean;
  canEdit: boolean;
  canArchive: boolean;
  onEdit: (r: RegisterRow) => void;
  onArchive: (r: RegisterRow) => void;
};
export function CollectionTable({
  rows,
  students,
  canEdit,
  canArchive,
  onEdit,
  onArchive,
}: Props) {
  const l = useLocale();
  return (
    <div
      className="table-wrap"
      role="region"
      aria-label={t(l, students ? "studentRegister" : "entries")}
      tabIndex={0}
    >
      <table className="collection-table">
        <thead>
          <tr>
            <th scope="col">{t(l, students ? "name" : "title")}</th>
            <th scope="col">{t(l, "status")}</th>
            {(canEdit || canArchive) && (
              <th scope="col" className="action-col">
                {t(l, "actions")}
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => {
            const label = rowName(r),
              known = ["open", "closed", "active", "archived"].includes(
                r.status,
              );
            return (
              <tr key={r.id}>
                <th scope="row">{label}</th>
                <td>
                  <span
                    className={
                      "badge " +
                      (r.status === "active" || r.status === "open"
                        ? "active"
                        : "")
                    }
                  >
                    <span aria-hidden="true" />
                    {known
                      ? t(
                          l,
                          r.status as "open" | "closed" | "active" | "archived",
                        )
                      : r.status}
                  </span>
                </td>
                {(canEdit || canArchive) && (
                  <td className="action-col">
                    {r.status !== "archived" && (
                      <div className="row-actions">
                        {canEdit && (
                          <button
                            className="icon-button"
                            aria-label={t(l, "edit") + " " + label}
                            title={t(l, "edit")}
                            onClick={() => onEdit(r)}
                          >
                            <Pencil size={16} />
                          </button>
                        )}
                        {canArchive && (
                          <button
                            className="icon-button"
                            aria-label={t(l, "archive") + " " + label}
                            title={t(l, "archive")}
                            onClick={() => onArchive(r)}
                          >
                            <Archive size={16} />
                          </button>
                        )}
                      </div>
                    )}
                  </td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
