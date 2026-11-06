# EduPilot OS frontend context and complete flows

Extends [the module screen matrix](frontend-matrix.md) and [tenancy specification](tenancy-and-placement.md). These are acceptance requirements for implementation, with implementation status tracked in the release board.

## Context and navigation

One context header shows the tenant/group, selected institution and optional campus, with unambiguous ancestry for duplicate names. Tenant switch, organization/school scope, campus/team filters, academic period and family child are different controls. The user can select only server-authorized choices. An explicitly labeled aggregate view means **all permitted schools**, not all schools or a place to create an ambiguously owned record. Create/publish/export/financial confirmations show the affected owning scope.

Routes and list filters carry stable IDs and harmless view state; no bearer, personal record fields or private draft belongs in a URL. Proposed route grammar is `#/t/:tenantId/o/:organizationId/s/:schoolId/modules/:moduleKey` with optional authorized narrower filters, or explicit tenant-level module routes for group-wide resources. The server remains authoritative regardless of URL shape. Unavailable deep links reveal no unrelated resource name.

Switch flow: request destination → resolve unsaved work in shared dialog → validate destination → retire old request epoch/cache/drafts/blobs/previews → mount new context → load allowed catalog/data. Until validation succeeds, retain the original heading/context. Cancel retains original editor and values. Writes capture the initiating scope and cannot be retargeted by a later selection. Old read results, write callbacks and 401s are discarded after an epoch change. Two tabs keep independent contexts; another tab's membership revocation still takes effect through server checks.

## Shared workspace composition

The canonical shell, heading/actions, form fields/error summary, record picker, table, detail sections, dialog, state panel, timeline and progress view own common behavior. Calendar and board views include agenda/list alternatives. Larger records use full-page editors rather than forcing every flow into a small modal. Business components receive typed data, permission decisions and command callbacks.

Supported configuration is a bounded versioned schema: navigation groups/order/pins, default route, density/width, allowed views, table column visibility/order/width, safe saved filter/sort/page size and detail section order. Personal settings apply only where policy permits. Shared presets have preview/publish/version history; personal reset and reset-to-inherited are distinct. Import/export contains layout identifiers/configuration only, never record values or privileges. Unknown/unsafe schema fields fail clearly. Hidden required fields become visible when validation points to them. Reorder controls have named up/down actions and keyboard/touch alternatives to dragging.

Shared configuration persists server-side per authorized scope/user where synchronization is required. Device-local settings are explicitly labeled; browser localStorage alone is not a cross-client sync mechanism. Every inherited setting shows effective value, source, locked reason and override/reset control where permitted. A team does not silently override an unrelated school policy; each setting defines applicable levels and precedence.

## Language and output

Separate interface language, entered content language, instruction/curriculum language and print/export language. A shared multilingual field editor shows language labels, required/fallback rules and missing/stale translations. Switching language preserves all unsaved translations. Names and official identifiers keep their original spelling. Published academic/financial documents snapshot wording, currency, timezone, locale and scope; later translations do not rewrite issued output.

Initial supported UI catalogs are English/French. A locale registry records fallback and direction; adding RTL or further catalogs requires layout/document/device acceptance before declaring support. School teaching/content languages are not restricted to UI language choices. Unicode, combining characters, apostrophes and long labels round-trip through APIs, imports, search, documents and clients.

## Complete transitions required per module flow

| Module family | Screens/actions beyond initial list/detail/editor |
|---|---|
| Tenancy / identity / users / permissions | Group onboarding, hierarchy/ancestry, invite accept/expired/revoked, password/session recovery, scoped grants/effective access, multiple memberships, last-admin protection |
| School settings / subscriptions | Inherited values, policy locks, effective diff/publish, reset-to-inherited; entitlement change/suspension, safe export and recovery |
| Academics / curriculum / classes | Year/term close/reopen impact; curriculum draft/review/publish/archive and school/cohort assignment; teaching/room assignment and capacity changes |
| Admissions / students | Application submit/status/missing-information/reject/reopen, duplicate decision, conversion; import upload→mapping→validation→accepted rows→commit→result/history |
| Guardians / enrollment | Relationship verification/consent/revoke; transfer eligibility; promotion preview/exceptions/commit/result; withdrawal reason and immutable history |
| Attendance / timetables | Missing marks, locked periods, correction request/approve; schedule conflict resolution, effective dates and publish preview |
| Learning / assessments / gradebook | Assignment resources/deadline/publish; submission revise/late states; teacher feedback publish; moderation/lock/unlock reasons and batch validation |
| Examinations / transcripts | Candidate register/withdraw, invigilator/room conflicts; issued-version preview, publish/revoke/reissue and old snapshot access |
| Discipline / scholarships | Case follow-up/assignment/close/reopen/disclosure; scholarship apply/review/reject/revoke and future-charge impact |
| Fees / billing | Schedule approval/effective date, installment/discount preview; invoice issue/void, credit allocation, aging and printable statements |
| Payments / accounting | Payment allocation, bank/provider reconciliation, refund approve/execute/result; chart of accounts, balanced journal post/reverse and fiscal close/reopen consequences |
| Staff / HR / payroll | Staff/contract lifecycle and private files; leave adjustment/request/cancel/review; payroll rules/exceptions/approve/export/result and locked payslip correction |
| Procurement / inventory / facilities | Order issue/amend/cancel, partial receive; location transfer dispatch/receive; stocktake variance approval; booking cancel and maintenance intake/assign/close |
| Library / transport | Checkout/renew/return/lost/damaged/reservation fulfillment; stop/vehicle/capacity editor; trip selection, boarding/reversal/history |
| Notifications / messaging / workflows | Language template preview, suppression/retry/dead-letter; participants/remove/moderate; definition version publish, delegate/reassign/escalate/cancel and decision reasons |
| Documents / audit | Upload→scan/quarantine→release, ownership/link scope, retention/delete; redacted audit detail and filtered authorized export |
| Portal / reporting / analytics | Multiple children across schools, consent, released output only; export queue/cancel/expiry/failure/history; accurate chart/table drill-down with scope-preserving back navigation |

Each implementation subplan binds these transitions to route IDs, actors/scopes, API commands/errors, cancel/back behavior and a seeded end-to-end scenario. A generic work-list page does not satisfy a transition.

## Tenant movement operator interface

Infrastructure controls are available only to authorized platform operators. Flow: pool/placement inventory → target selection → preflight and affected-tenant preview → backup/schema/capacity/region findings → maintenance/start confirmation → resumable operation detail → validation/cutover/activation/observation → completed history. Show last confirmed update time and server state. Never expose connection strings or credentials.

The job ID survives reconnect/reopen; unknown start outcomes look up the same operation before retry. Double start cannot create a second move. Cancel is shown only before its permitted boundary. After cutover, recovery invokes the server's recovery/reverse-move contract; no misleading one-click stale-source rollback. Ordinary tenant users see a localized maintenance message and retry guidance with no silent offline mutation queue. Switching campus or organization never selects a physical database.

## Client action parity

| Role/workflow | Web/Wails | Expo release requirement |
|---|---|---|
| Group/platform administrator | Full hierarchy, configuration, access, move operations and reports | Context/account/security and read-only permitted operational summary; complex administration clearly handed off to web |
| Teacher | Full class, timetable, assignment, attendance and grade flows | Assigned roster, attendance/corrections, timetable, assignments/submissions/feedback, messaging and permitted files |
| Finance/HR/operations administrator | Full approved administrative workflows | Assigned approval inbox, notifications and safe read-only summaries; accounting/payroll configuration on web |
| Student | Own timetable, assignments/submissions, published grades, messages/files | Same daily actions, secure file upload/download, session expiry and online-only command feedback |
| Guardian | Linked children, published records, statements, consent and communication | Same family actions with explicit child/school context; no staff-only data |
| Transport staff | Trip roster/boarding and corrections | Assigned trip selection, boarding/reversal and current online status; no undocumented offline sensitive cache |

No mobile control opens an unfinished dead end. Each handoff names the web action and retains only safe intended navigation, never puts credentials into a link. Push notification receipt does not itself authorize record access.

## Professional quality evidence

At 320px and 200% zoom ordinary navigation/forms have no page-level horizontal overflow. Complex grids use a named contained scroll region and a complete alternative. Sticky actions and the on-screen keyboard do not cover focus/errors. Use visible focus, native semantics, explicit status text, adequate contrast, reduced-motion/forced-color support and usable touch targets. On closing the mobile menu without navigation, restore its toggle focus; on route change, focus the new heading/main. Dialogs trap and restore focus predictably.

Record role, context fixture, viewport, locale/theme, keyboard result, screenshots and browser assertions per accepted flow. Include failed API, slow/offline, conflict, revocation, stale callback and move-maintenance cases. Compare sibling pages to ensure token/layout/feedback consistency. A visual capture complements functional evidence; it does not replace it.
