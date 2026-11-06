# EduPilot OS full frontend delivery matrix

This matrix is a release checklist, not a declaration that every screen already exists. Each domain story requires a complete real-data page flow. Web and Wails share these pages; mobile parity is specified by CLI-02/03 in the release plan.

Binding additions: [context, inherited customization and complete business flows](frontend-context-and-flows.md), [tenancy/placement contract](tenancy-and-placement.md). These include group hierarchy, context switching, multilingual content and operator move screens.

## Shared professional UI acceptance

Every row inherits: role-aware navigation; shared shell/heading/actions; localized labels and validation; accessible list/detail/editor as applicable; loading, empty, no-results, error/retry, forbidden, offline, dirty and stale-version states; keyboard focus restoration; responsive layout at 320px, tablet and desktop; 200% zoom; all themes and long English/French text. Lists use bounded server pagination and safe URL filters. A disabled or unfinished domain workflow is never presented as a working action.

Reusable templates: record collection/detail/editor, settings sections, roster/grid with list alternative, calendar with agenda alternative, approval board with list alternative, financial ledger and printable report. Build a template when its first real story needs it, then reuse it.

## Per-module delivery

| Story | Module / route root | Required screens and flow | Current domain UI status |
|---|---|---|---|
| TEN-01 | tenancy · `#/modules/tenancy` | Institution profile, lifecycle confirmation and read-only placement diagnostics | Planned; current screen is a workspace work list |
| IAM-01 | identity · `#/modules/identity` | Sign-in, password recovery/change, session list with revoke controls | Planned; current screen is a workspace work list |
| IAM-02 | users · `#/modules/users` | Searchable member directory, add member/invite, profile, deactivate confirmation | Planned; current screen is a workspace work list |
| IAM-03 | permissions · `#/modules/permissions` | Role editor, effective-access preview, assignment workflow | Planned; current screen is a workspace work list |
| CFG-01 | school_settings · `#/modules/school_settings` | Shared settings sections, clear validation, save/reset and conflict recovery | Planned; current screen is a workspace work list |
| SUB-01 | subscriptions · `#/modules/subscriptions` | Plan/usage/renewal status and administrator entitlement controls | Planned; current screen is a workspace work list |
| ACD-01 | academics · `#/modules/academics` | Year/term list, editor, calendar summary and close confirmation | Planned; current screen is a workspace work list |
| CUR-01 | curriculum · `#/modules/curriculum` | Subject catalog, curriculum detail and weight editor | Planned; current screen is a workspace work list |
| CLS-01 | classes · `#/modules/classes` | Class directory, roster, staff assignment and capacity indicators | Planned; current screen is a workspace work list |
| ADM-01 | admissions · `#/modules/admissions` | Intake form, review board/list, applicant detail and decision dialog | Planned; current screen is a workspace work list |
| STU-01 | students · `#/modules/students` | Full student list/detail with profile, enrollment, guardians and history tabs; import preview | Partial: list/create/edit/archive; full profile/import still planned |
| GUA-01 | guardians · `#/modules/guardians` | Family directory, relationship editor and linked children view | Planned; current screen is a workspace work list |
| ENR-01 | enrollment · `#/modules/enrollment` | Enrollment roster, placement wizard, transfer and promotion preview | Partial: placements and withdrawal; class/transfer/promotion planned |
| ATT-01 | attendance · `#/modules/attendance` | Date/class roster, keyboard mark entry, review/submit and correction history | Planned; current screen is a workspace work list |
| TTM-01 | timetables · `#/modules/timetables` | Weekly calendar plus accessible list alternative, slot editor and conflict panel | Planned; current screen is a workspace work list |
| LRN-01 | learning · `#/modules/learning` | Teacher assignment editor, student task list/submission and feedback detail | Planned; current screen is a workspace work list |
| ASM-01 | assessments · `#/modules/assessments` | Assessment list/detail, setup form, publish checklist | Planned; current screen is a workspace work list |
| GRD-01 | gradebook · `#/modules/gradebook` | Accessible marks grid with non-grid alternative, totals and conflict resolution | Planned; current screen is a workspace work list |
| EXM-01 | examinations · `#/modules/examinations` | Session scheduler, candidate/room allocation and printable roster | Planned; current screen is a workspace work list |
| TRN-01 | transcripts · `#/modules/transcripts` | Report preview, moderation queue, publish dialog and accessible print/download | Planned; current screen is a workspace work list |
| DSC-01 | discipline · `#/modules/discipline` | Case list/detail with timeline, action editor and restricted disclosure | Planned; current screen is a workspace work list |
| SCH-01 | scholarships · `#/modules/scholarships` | Award catalog, application review and remaining-allocation view | Planned; current screen is a workspace work list |
| FEE-01 | fees · `#/modules/fees` | Schedule editor, installment table and student charge preview | Planned; current screen is a workspace work list |
| BIL-01 | billing · `#/modules/billing` | Invoice list/detail/editor, issue confirmation, credit note and statement print | Planned; current screen is a workspace work list |
| PAY-01 | payments · `#/modules/payments` | Receipt entry, payment detail, unmatched-payment queue and refund approval | Planned; current screen is a workspace work list |
| ACC-01 | accounting · `#/modules/accounting` | Account ledger, journal editor/review/post, trial balance and close checklist | Planned; current screen is a workspace work list |
| STF-01 | staff · `#/modules/staff` | Staff directory/detail, profile editor and assignment summary | Planned; current screen is a workspace work list |
| HR-01 | hr · `#/modules/hr` | HR profile, leave calendar/request/review and document access | Planned; current screen is a workspace work list |
| PRL-01 | payroll · `#/modules/payroll` | Run wizard, exceptions, payroll review, approval and payslip access | Planned; current screen is a workspace work list |
| PRC-01 | procurement · `#/modules/procurement` | Supplier list, requisition editor, approval inbox and receipt form | Planned; current screen is a workspace work list |
| INV-01 | inventory · `#/modules/inventory` | Item list/detail, stock movement forms and stocktake reconciliation | Planned; current screen is a workspace work list |
| FAC-01 | facilities · `#/modules/facilities` | Space/asset directory, booking calendar/list and maintenance board | Planned; current screen is a workspace work list |
| LIB-01 | library · `#/modules/library` | Catalog/copy detail, checkout/return desk, reservations and overdue list | Planned; current screen is a workspace work list |
| TRP-01 | transport · `#/modules/transport` | Route editor, rider roster, vehicle assignment and mobile boarding checklist | Planned; current screen is a workspace work list |
| NTF-01 | notifications · `#/modules/notifications` | Notification inbox, preference settings and operator delivery diagnostics | Planned; current screen is a workspace work list |
| MSG-01 | messaging · `#/modules/messaging` | Inbox/thread/composer, recipient picker and delivery state | Planned; current screen is a workspace work list |
| WFL-01 | workflows · `#/modules/workflows` | Workflow definition editor, assigned-task inbox and instance timeline | Planned; current screen is a workspace work list |
| DOC-01 | documents · `#/modules/documents` | Document library, upload with progress, preview/download, details and delete confirmation | Partial: upload/download by ID; full library/metadata planned |
| AUD-01 | audit · `#/modules/audit` | Audit table/detail, safe filters and authorized export | Planned; current screen is a workspace work list |
| POR-01 | portal · `#/modules/portal` | Family child switcher, student home, attendance/results/fees/documents pages | Planned; current screen is a workspace work list |
| RPT-01 | reporting · `#/modules/reporting` | Report catalog, parameter form, preview and export jobs/history | Planned; current screen is a workspace work list |
| ANL-01 | analytics · `#/modules/analytics` | Real KPI summaries, accessible charts/tables, filters and explanatory empty states | Planned; current screen is a workspace work list |

## Cross-cutting screen flows

| Flow | Screens | Evidence required |
|---|---|---|
| Identity | Sign-in, password recovery/reset/change, active sessions | Invalid/expired credentials, throttling, revocation, paste/password-manager behavior |
| Navigation | Directory, pinned/reordered workspaces, scoped home, breadcrumb, unavailable route | Keyboard/narrow navigation; safe deep links; correct title and focus |
| Preferences | Locale, theme, accent, density, width, pins/order and reset | Per-user/school isolation; live application across pages; storage-denied feedback |
| Recovery | Error boundary, offline status, retry, expired session, unsaved-change dialog | Draft retention; no duplicate write replay; no stale-user data |
| Operations | Health/configuration diagnostics, backup/restore and queue status where granted | No secrets disclosed; real state and actionable failures |

## Evidence recording

For each accepted row record commit, tested roles, desktop/narrow captures, keyboard result, locale/theme coverage, browser cases and any unavailable hardware. A screenshot alone does not prove that its API or permissions work. No screen is marked complete until the corresponding domain acceptance journey passes.
