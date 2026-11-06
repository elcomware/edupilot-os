# EduPilot OS · administrative interface

The school register is our model: clear section tabs, calm writing surfaces and room for names. The module directory, student register and other workspaces are for staff returning to a task, often on a small screen or in a second language. The distinctive element is the **index tab**: a cropped, stepped page mark repeated in the navigation and at the sign-in desk. Do not add fabricated activity, school statistics, welcome cards or financial transactions.

## Shared vocabulary

| Role | Source | Use |
|---|---|---|
| Ink, paper, surface, line, muted | `clients/web/src/styles/tokens.css` | Text, reading plane, controls and rules |
| Pine, indigo, clay | Same tokens; chosen via `data-accent` | Selected navigation, primary actions and links |
| Theme, density and reading width | Same tokens; scoped preference store | All application pages, including desktop |
| Display, body, utility | Same tokens | Georgia headings, system sans controls, platform mono identifiers |
| Structure | Shell, PageHeading, StatePanel, Dialog, CollectionTable | One navigation frame, visible heading hierarchy, simple tables and reusable forms |

Student registers, work lists and enrollment placements supply typed columns, rows and actions to `components/WorkspaceTable.tsx`. Edit its disclosure and table markup once to update all three. Display labels come from the active catalog and language.

Runtime CSS is canonical. Its named tokens define light and dark palettes; user preferences change semantic roles, never isolated controls. Use the primary accent once per action group. Long French labels wrap, name columns are allowed to expand, and keyboard focus stays visible. Native selects are intentional. Icons accompany words unless individually named.

The directory is a grouped index, not a dashboard of unmeasured trends. Group order and accessible records come from the API. Student rows have real names, status and archival actions. Other modules share a clearly labeled workspace-entry list; new domain screens replace that adapter only when corresponding use cases exist. Web and Wails use the same React shell. Expo shares the contract but uses native interaction patterns.

## Maintainers

Update the runtime token and this rationale together for cross-screen identity changes. Keep translation keys in the app dictionary. Add a new module by registering it in the server catalog; do not fork the shell or author an identical list screen. See UX-CONTRACT.md for observable behavior.
