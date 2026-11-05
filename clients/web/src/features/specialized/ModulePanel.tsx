import type { Principal } from "@edupilot/api-client";
import { EnrollmentPanel } from "./EnrollmentPanel";
import { DocumentsPanel } from "./DocumentsPanel";
import "../../styles/panels.css";
// Specialized workflows are opt-in. All other modules share the entries collection.
const panels = { enrollment: EnrollmentPanel, documents: DocumentsPanel };
export function ModulePanel({
  moduleKey,
  principal,
}: {
  moduleKey: string;
  principal: Principal;
}) {
  const Panel = panels[moduleKey as keyof typeof panels];
  return Panel ? <Panel principal={principal} /> : null;
}
