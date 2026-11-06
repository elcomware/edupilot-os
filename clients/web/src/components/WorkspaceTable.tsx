import { useState, type ReactNode } from "react";
import { t } from "../app/copy";
import { useLocale, usePreferences } from "../app/preferences";

export type Column<Row> = {
  key: string;
  label: string;
  render: (row: Row) => ReactNode;
  rowHeader?: boolean;
  optional?: boolean;
  className?: string;
};
type Props<Row> = {
  view: string;
  label: string;
  rows: Row[];
  rowKey: (row: Row) => string;
  columns: Column<Row>[];
};

// Pages own labels, data and permitted actions. This shared table owns layout
// and persists only the public names of optional columns.
export function WorkspaceTable<Row>({
  view,
  label,
  rows,
  rowKey,
  columns,
}: Props<Row>) {
  const locale = useLocale();
  const layouts = usePreferences((s) => s.value.columns);
  const stored = usePreferences((s) => s.stored);
  const setColumns = usePreferences((s) => s.setColumns);
  const [failedView, setFailedView] = useState<string | null>(null);
  const hidden = layouts[view] ?? [];
  const options = columns.filter(
    (column) => column.optional && !column.rowHeader,
  );
  const visible = columns.filter(
    (column) => !options.includes(column) || !hidden.includes(column.key),
  );
  return (
    <div className="workspace-table">
      {options.length > 0 && (
        <details className="table-columns">
          <summary>{t(locale, "columns")}</summary>
          <fieldset>
            <legend>{t(locale, "shownColumns")}</legend>
            {options.map((column) => (
              <label key={column.key}>
                <input
                  type="checkbox"
                  checked={!hidden.includes(column.key)}
                  onChange={(event) => {
                    setColumns(
                      view,
                      event.target.checked
                        ? hidden.filter((key) => key !== column.key)
                        : [...hidden, column.key],
                    );
                    setFailedView(
                      usePreferences.getState().stored ? null : view,
                    );
                  }}
                />
                {column.label}
              </label>
            ))}
          </fieldset>
        </details>
      )}
      <p role="status" className="save-message table-save-status">
        {!stored && failedView === view ? t(locale, "storageUnavailable") : ""}
      </p>
      <div className="table-wrap" role="region" aria-label={label} tabIndex={0}>
        <table className="collection-table" aria-label={label}>
          <thead>
            <tr>
              {visible.map((column) => (
                <th key={column.key} scope="col" className={column.className}>
                  {column.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={rowKey(row)}>
                {visible.map((column) =>
                  column.rowHeader ? (
                    <th
                      key={column.key}
                      scope="row"
                      className={column.className}
                    >
                      {column.render(row)}
                    </th>
                  ) : (
                    <td key={column.key} className={column.className}>
                      {column.render(row)}
                    </td>
                  ),
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
