import type { Locale } from "@edupilot/api-client";
const labels: Record<string, readonly [string, string]> = {
  title: ["My account", "Mon compte"],
  role: ["Role", "Rôle"],
  permissions: ["Permissions", "Autorisations"],
  admin: ["School administrator", "Administrateur de l’établissement"],
  registrar: ["Registrar", "Gestionnaire des inscriptions"],
  viewer: ["Viewer", "Lecteur"],
  "modules.read": ["View the directory", "Consulter le répertoire"],
  "workspace.read": [
    "View school entries",
    "Consulter les entrées de l’établissement",
  ],
  "workspace.write": [
    "Edit school entries",
    "Modifier les entrées de l’établissement",
  ],
  "auth.members.manage": ["Manage members", "Gérer les membres"],
};
export function accountLabel(locale: Locale, key: string) {
  return labels[key]?.[locale === "fr" ? 1 : 0] ?? key;
}
