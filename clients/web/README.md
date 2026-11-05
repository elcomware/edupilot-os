# Web and desktop workspace

This React application uses the same versioned HTTP contract as Expo. Vite compiles it to static assets in `clients/web/dist`. Its relative asset paths and fragment routes also work when packaged by Wails. Browser and desktop use the same screens; no business rule lives in the shell.

From the repository root, install the npm workspaces, then use `npm run dev:web`. The dev server proxies `/api` to Goravel on port 3000; override `EDUPILOT_API_PROXY` when necessary. In production the default API root is same-origin `/api/v1`. For a separate allowed origin, set the public build-time `VITE_API_BASE_URL` to the full versioned HTTPS URL. Never put credentials in Vite environment variables. `npm run build:web` type-checks and builds the web bundle; `npm run test:web` runs its unit tests.

| Route | Behavior |
|---|---|
| `#/workspace` | Catalog grouped by school capability; search, filter, pin |
| `#/modules/students` | Active student register; create, edit, archive |
| `#/modules/<registered-key>` | Catalog-authorized module workspace; list, create and update entries |
| `#/settings` | Per-school/per-principal language, appearance, accent, density and reading width |

Session tokens live in memory only. Refresh or reopen the app and sign in again. The current fragment is retained for deep links. Server-side authentication and authorization apply to every request. Theme preferences and pinned module keys persist on this device; no student data or credentials are stored locally.

Extend `features/collections` for a new specialized experience. Keep generic workspace entries labeled as entries. Shared transport and DTOs live in `packages/api-client`, while query state belongs to TanStack Query and cosmetic controls belong to Zustand. See the root DESIGN.md and UX-CONTRACT.md.
