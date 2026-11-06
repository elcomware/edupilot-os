# EduPilot OS full frontend delivery matrix

This matrix is a release checklist, not a declaration that every screen already exists. Each domain story requires a complete real-data page flow. Web and Wails share these pages; mobile parity is specified by CLI-02/03 in the release plan.

Binding additions: [context, inherited customization and complete business flows](frontend-context-and-flows.md), [tenancy/placement contract](tenancy-and-placement.md). These include group hierarchy, context switching, multilingual content and operator move screens.

## Shared professional UI acceptance

Every row inherits: role-aware navigation; shared shell/heading/actions; localized labels and validation; accessible list/detail/editor as applicable; loading, empty, no-results, error/retry, forbidden, offline, dirty and stale-version states; keyboard focus restoration; responsive layout at 320px, tablet and desktop; 200% zoom; all themes and long English/French text. Lists use bounded server pagination and safe URL filters. A disabled or unfinished domain workflow is never presented as a working action.

Reusable templates: record collection/detail/editor, settings sections, roster/grid with list alternative, calendar with agenda alternative, approval board with list alternative, financial ledger and printable report. Build a template when its first real story needs it, then reuse it.

## Per-module delivery

| Story | Module / route root | Required screens and flow | Current domain UI status |
|---|---|---|---|
| TEN-01 | tenancy · `#/modules/tenancy` | Institution profile, lifecycle confirmation and read-only placement diagnostics | Planned |
| IAM-01 | auth · `#/modules/auth` | Sign-in, password recovery/change, session list with revoke controls | My account displays the current role and grants. |
| IAM-02 | auth · `#/modules/auth` | Searchable member directory, add member/invite, profile, deactivate confirmation | Planned within Auth. |
| IAM-03 | auth · `#/modules/auth` | Role editor, effective-access preview, assignment workflow | Planned within Auth. |
| CFG-01 | school_settings · `#/modules/school_settings` | Shared settings sections, clear validation, save/reset and conflict recovery | Planned |
| SUB-01 | subscriptions · `#/modules/subscriptions` | Plan/usage/renewal status and administrator entitlement controls | Planned |
| ACD-01 | academics · `#/modules/academics` | Year/term list, editor, calendar summary and close confirmation | Planned |
| CUR-01 | curriculum · `#/modules/curriculum` | Subject catalog, curriculum detail and weight editor | Planned |
| CLS-01 | classes · `#/modules/classes` | Class directory, roster, staff assignment and capacity indicators | Planned |
| ADM-01 | admissions · `#/modules/admissions` | Intake form, review board/list, applicant detail and decision dialog | Planned |
| STU-01 | students · `#/modules/students` | Full student list/detail with profile, enrollment, guardians and history tabs; import preview | Planned |
| GUA-01 | guardians · `#/modules/guardians` | Family directory, relationship editor and linked children view | Planned |
| ENR-01 | enrollment · `#/modules/enrollment` | Enrollment roster, placement wizard, transfer and promotion preview | Planned |
| ATT-01 | attendance · `#/modules/attendance` | Date/class roster, keyboard mark entry, review/submit and correction history | Planned |
| TTM-01 | timetables · `#/modules/timetables` | Weekly calendar plus accessible list alternative, slot editor and conflict panel | Planned |
| LRN-01 | learning · `#/modules/learning` | Teacher assignment editor, student task list/submission and feedback detail | Planned |
| ASM-01 | assessments · `#/modules/assessments` | Assessment list/detail, setup form, publish checklist | Planned |
| GRD-01 | gradebook · `#/modules/gradebook` | Accessible marks grid with non-grid alternative, totals and conflict resolution | Planned |
| EXM-01 | examinations · `#/modules/examinations` | Session scheduler, candidate/room allocation and printable roster | Planned |
| TRN-01 | transcripts · `#/modules/transcripts` | Report preview, moderation queue, publish dialog and accessible print/download | Planned |
| DSC-01 | discipline · `#/modules/discipline` | Case list/detail with timeline, action editor and restricted disclosure | Planned |
| SCH-01 | scholarships · `#/modules/scholarships` | Award catalog, application review and remaining-allocation view | Planned |
| FEE-01 | fees · `#/modules/fees` | Schedule editor, installment table and student charge preview | Planned |
| BIL-01 | billing · `#/modules/billing` | Invoice list/detail/editor, issue confirmation, credit note and statement print | Planned |
| PAY-01 | payments · `#/modules/payments` | Receipt entry, payment detail, unmatched-payment queue and refund approval | Planned |
| ACC-01 | accounting · `#/modules/accounting` | Account ledger, journal editor/review/post, trial balance and close checklist | Planned |
| STF-01 | staff · `#/modules/staff` | Staff directory/detail, profile editor and assignment summary | Planned |
| HR-01 | hr · `#/modules/hr` | HR profile, leave calendar/request/review and document access | Planned |
| PRL-01 | payroll · `#/modules/payroll` | Run wizard, exceptions, payroll review, approval and payslip access | Planned |
| PRC-01 | procurement · `#/modules/procurement` | Supplier list, requisition editor, approval inbox and receipt form | Planned |
| INV-01 | inventory · `#/modules/inventory` | Item list/detail, stock movement forms and stocktake reconciliation | Planned |
| FAC-01 | facilities · `#/modules/facilities` | Space/asset directory, booking calendar/list and maintenance board | Planned |
| LIB-01 | library · `#/modules/library` | Catalog/copy detail, checkout/return desk, reservations and overdue list | Planned |
| TRP-01 | transport · `#/modules/transport` | Route editor, rider roster, vehicle assignment and mobile boarding checklist | Planned |
| NTF-01 | notifications · `#/modules/notifications` | Notification inbox, preference settings and operator delivery diagnostics | Planned |
| MSG-01 | messaging · `#/modules/messaging` | Inbox/thread/composer, recipient picker and delivery state | Planned |
| WFL-01 | workflows · `#/modules/workflows` | Workflow definition editor, assigned-task inbox and instance timeline | Planned |
| DOC-01 | documents · `#/modules/documents` | Document library, upload with progress, preview/download, details and delete confirmation | Planned |
| AUD-01 | audit · `#/modules/audit` | Audit table/detail, safe filters and authorized export | Planned |
| POR-01 | portal · `#/modules/portal` | Family child switcher, student home, attendance/results/fees/documents pages | Planned |
| RPT-01 | reporting · `#/modules/reporting` | Report catalog, parameter form, preview and export jobs/history | Planned |
| ANL-01 | analytics · `#/modules/analytics` | Real KPI summaries, accessible charts/tables, filters and explanatory empty states | Planned |

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


The release board tracks verification of the installed foundation.
