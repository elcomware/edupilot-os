# Shared client resources

Core owns reusable UI at the repository root. Auth owns its sign-in and account pages. Tenancy exposes its school workspace through the same registration pattern.

| Source | Consumers |
|---|---|
| `js/layouts/WelcomeLayout.tsx` | Auth sign-in on web and Wails |
| `js/layouts/TenantAdminLayout.tsx` | Signed-in school workspace |
| `js/layouts/PlatformAdminLayout.tsx` | Frame for future operator pages. The host supplies authorized navigation. |
| `js/components/` and `css/` | Buttons, cards, skeletons, tables, dialogs and shared theme tokens |
| `native/` | Native controls, server-state handling, session lifecycle and localization |
| `js/pages.web.ts` | Vite resolves explicit module imports at build time. Wails embeds `clients/web/dist`. |
| `js/pages.native.ts` | Expo/Metro resolves the native imports into Android and iOS bundles. |

Register each module’s target-specific exports in the appropriate maps. TypeScript checks page props. Clients use the signed-in user’s server catalog to select accessible pages.
