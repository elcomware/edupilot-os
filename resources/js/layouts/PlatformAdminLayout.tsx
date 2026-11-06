import type { ReactNode } from "react";
import { AdminLayout } from "./AdminLayout";
// The platform host supplies authorized navigation and a localized scope label.
export function PlatformAdminLayout(props: {
  navigation: ReactNode;
  scope: string;
  title: string;
  children: ReactNode;
}) {
  return <AdminLayout {...props} />;
}
