# EduPilot OS · administrative interface

EduPilot OS uses the approved gold EP monogram and outlined wordmark on garnet navigation surfaces. The sign-in page gives the mark room at desktop widths and uses a compact header on narrow screens. The staff workspace uses warm paper and porcelain surfaces for legibility in English and French.

## Shared vocabulary

| Role | Source | Use |
|---|---|---|
| Ink, paper, surface, line, muted | `modules/core/resources/css/tokens.css` | Text, reading plane, controls and rules |
| Garnet (default), pine, indigo, clay | Same tokens; chosen via `data-accent` | Selected navigation, primary actions and links |
| Theme, density and reading width | Same tokens; scoped preference store | All application pages, including desktop |
| Display, body, utility | Same tokens | Georgia headings, system sans controls, platform mono identifiers |
| Structure | Shell, PageHeading, StatePanel, Dialog, CollectionTable | One navigation frame, visible heading hierarchy, simple tables and reusable forms |

School work lists supply columns, rows and permitted actions to the shared WorkspaceTable.

Core owns the brand palette in `modules/core/resources/brand/palette.json`. Web and desktop consume its generated CSS and Expo consumes the JSON directly. The named tokens define light and dark palettes; user preferences change semantic roles, never isolated controls. Use the primary accent once per action group. Long French labels wrap, name columns are allowed to expand, and keyboard focus stays visible. Native selects are intentional. Icons accompany words unless individually named.

The directory is a grouped index, not a dashboard of unmeasured trends. Group order and accessible records come from the API. The account page shows the user’s school, role and permissions. Schools has a versioned work list. Web and Wails use the same React shell. Expo shares the contract but uses native interaction patterns.

## Maintainers

Update the runtime token and this rationale together for cross-screen identity changes. Keep translation keys in the app dictionary. Add a new module by registering it in the server catalog; do not fork the shell or author an identical list screen. See UX-CONTRACT.md for observable behavior.
