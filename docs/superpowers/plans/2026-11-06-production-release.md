# EduPilot OS Production Release Implementation Plan

**Reference module decision.** Auth is the reference for new work. Use the [module convention and relationship walkthrough](../../../modules/auth/README.md). Use the `auth` catalog key. Users, roles and permissions belong to Auth.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking. Continue within the user's existing implementation and push authorization.

**Goal:** Deliver the complete EduPilot OS product, including a professionally finished frontend, verified domain workflows and an operable native production release.

**Architecture:** Goravel supplies the service container, Auth guard, Schema builder, cache and logging. Modules own models, services and migrations and publish events through Core. React provides shared web/Wails layouts; Expo uses the same typed API for mobile tasks.

**Tech Stack:** Go/Goravel 1.18, SQLite/PostgreSQL, React/TypeScript, TanStack Query, Zustand, Vite, Wails, Expo, Vitest, Playwright and Go testing. Pin existing compatible dependencies unless a verified update is required.

**Spec:** [Production scope](../../product/production-scope.md), [Tenancy and placement](../../product/tenancy-and-placement.md), [DESIGN.md](../../../DESIGN.md), [UX-CONTRACT.md](../../../UX-CONTRACT.md).

## Global constraints

- The product is **EduPilot OS**, the GitHub repository is **elcomware/edupilot-os**, and the root Go module is `github.com/elcomware/edupilot`.
- Native installation and operation are the documented default. Docker is optional and has explicit `docker-*` commands.
- Use Goravel module folder names. Keep business rules in module models and services. Connect feature modules through Core contracts at bootstrap or through events.
- Tenant, organization, school, campus, location, team and curriculum are distinct scoped concepts. Tenant placement supports named shared/dedicated pools and controlled tenant moves.
- Every shipped module includes its real backend workflows, complete UI, permissions, migrations, tests and operator documentation.
- Use shared layout, form, table, dialog and feedback components; domain pages provide typed data and actions.
- No fabricated activity, counts, transactions or charts. No credentials or sensitive records in persisted cosmetic preferences or URLs.
- English/French, responsive web/desktop, role-appropriate mobile, keyboard operation and WCAG 2.2 AA are release requirements.
- Run available tests; report skips by name, reason and dependency. Unavailable verification keeps its gate open.
- Push the plan first, then each coherent checkpoint. Do not deploy to production or send external messages under code-writing authorization.

## Review focus

1. Tenant or permission changes during an open editor: writes must fail safely and previous-user caches/drafts must not appear in the next session (S03, S20).
2. Concurrent or retried writes: no duplicate placement, payment, attendance or notification; optimistic conflicts preserve the user's work (S06, S08, S11, S15).
3. Narrow viewport, 200% zoom, long French text and keyboard-only input: dialogs, actions and errors remain reachable (S02 and every UI story).
4. Native startup, migration or worker failure: process exits nonzero, healthy siblings are stopped, diagnostics omit secrets, restart is safe (S01, S18).
5. Late callbacks, offline timeouts and revoked sessions: no stale data exposure, silent write replay or false success message (S02, S03, S19).

## Scrum working agreement

Use the backlog below as the product backlog. Each row has a stable story ID, dependency, deliverable and acceptance test. Sprints are ordered delivery increments, not promises about calendar duration. Forecast points are relative complexity (3 small, 5 moderate, 8 substantial, 13 split before execution); they are not hours. Refine each upcoming sprint into file-level RED/GREEN tasks before implementation. Do not mark a sprint done because its scaffolding exists.

Roles: product owner = repository owner; implementation/QA = agent with independent code review; release approver = deployment operator. User authorization already covers planning, implementation, testing and pushes. Review/retro is recorded in [release status](../../release/status.md), without asking for permission to continue routine work.

**Definition of ready:** story has actors, permission keys, fields, state transitions, API/error contract, page flow, test fixtures and dependencies. **Definition of done:** rules and migrations implemented; UI wired to real data; negative/concurrency/tenant tests green; API/reference docs updated; accessibility and responsive checks recorded; reviewer findings resolved; commit pushed. A provider or hardware gate can remain blocked, but the story cannot be called fully done if that gate is required.

The working board uses Planned → Ready → In progress → Review → Verified → Released, with Blocked for named external requirements. Track commit/evidence links, not percentages based on file counts. At each increment record delivered stories, failed/skipped checks, decisions and next dependency. Re-estimate after the first baseline and after each domain integration.

## Ordered sprint / epic map

| Sprint | Epic | Stories | Depends on | Demonstrable increment |
|---|---|---|---|---|
| S01 | E01 Native foundation and trustworthy verification | OPS-01..04 | Foundation | Fresh native SQLite install, branded app, test baseline, optional containers |
| S02 | E21 Group tenancy foundation + E02 frontend system | TEN-02..04, ORG-01..03, CTX-01..02, UI-01..08, LNG-01..02 | S01 | Tenant/school separation, scoped hierarchy, context UI, shared responsive page system, inherited workspaces and multilingual records |
| S03 | E03 Institution and access administration | TEN-01, IAM-01..03, CFG-01..02, SUB-01 | S01–02 | School setup, members, scoped roles, sessions and subscription entitlements UI |
| S04 | E04 Academic configuration | ACD-01, CUR-01, CLS-01 | S03 | Year/term, curriculum and class configuration with usable administration screens |
| S05 | E05 Admissions and family records | ADM-01, STU-01, GUA-01 | S03–04 | Application → admission → student + family record, duplicate handling |
| S06 | E06 Enrollment and attendance | ENR-01, ATT-01 | S04–05 | Enrollment progression, roster and teacher attendance flow |
| S07 | E07 Scheduling and learning | TTM-01, LRN-01 | S04, S06 | Conflict-aware timetable and assignment/submission flows |
| S08 | E08 Assessment and academic publication | ASM-01, GRD-01, EXM-01, TRN-01 | S06–07 | Validated marks → moderated grades → published transcript |
| S09 | E09 Student support and discipline | DSC-01, SCH-01 | S05–06 | Restricted case handling and scholarship approvals |
| S10 | E10 Fees and billing | FEE-01, BIL-01 | S05–06, S09 | Fee schedules, student invoices, credit notes and statements |
| S11 | E11 Payments and accounting | PAY-01, ACC-01 | S10 | Idempotent receipts, reconciliation, balanced journal and close |
| S12 | E12 Staff and payroll | STF-01, HR-01, PRL-01 | S03, S11 | Staff records, leave, approved payroll runs and exports |
| S13 | E13 Purchasing and resources | PRC-01, INV-01, FAC-01 | S11–12 | Requisition/approval/receipt, stock and room/asset workflows |
| S14 | E14 Library and transport | LIB-01, TRP-01 | S05, S12–13 | Lending/returns and routes/vehicle/boarding operations |
| S15 | E15 Communication and automation | NTF-01, MSG-01, WFL-01 | S03, S05–14 | Preferences, deliveries, scoped messages and resumable approvals |
| S16 | E16 Documents, audit and family portal | DOC-01, AUD-01, POR-01 | S05–15 | Searchable authorized files, immutable audit search and family portal |
| S17 | E17 Reporting and analytics | RPT-01, ANL-01 | S08, S11, S16 | Accurate permission-filtered reports, exports and drill-down dashboards |
| S18 | E18 Reliability and E22 tenant mobility | MOV-01..06, OPS-05..09 | TEN-02..04, S01–17 | Native packages/services, shared/isolated tenant moves, backup/restore, monitoring, upgrade and recovery drills |
| S19 | E19 Client completion | CLI-01..04 | S02–17 | Full shared desktop UI, complete supported mobile flows and device/browser QA |
| S20 | E20 Release qualification | REL-01..06 | All | Security/accessibility/performance review, UAT and signed release candidate |

## Expanded tenancy, organization, customization and mobility backlog

These are first-class stories added by the owner's group-tenancy clarification, not hidden inside a small settings story. S02 is a multi-slice foundation epic; split into sprint-sized vertical increments after baseline measurement. Tenant means the isolation/customer boundary; school is a child academic institution. The architecture specification is authoritative on hierarchy, scope, fencing and recovery.

| Story / pts | Dependency | Deliverable including full UI | Acceptance |
|---|---|---|---|
| TEN-02 / 8 | S01 | Distinct typed tenant/organization/school/campus/location/team context and repository ownership; schema/ID inventory | Two schools in one tenant and two tenants in one pool stay distinct; no implicit admin bypass |
| TEN-03 / 8 | TEN-02 | Recover and verify the foundation on disposable shared and dedicated pools | Restore every scoped record and grant, reject ambiguous placement, and verify interruption and retry |
| TEN-04 / 8 | TEN-03 | Durable pool registry/placement epoch and shared TenantUnitOfWork boundary for HTTP/jobs/files/exports | Named shared pools and dedicated pool route correctly; stale epoch fails; no silent shared fallback; control/data split supported |
| ORG-01 / 8 | TEN-03, IAM-03 | Organization/school/campus hierarchy directories/detail/edit with membership and lifecycle preview | Same-named schools distinguished by ancestry; cycles/cross-tenant parents rejected; reparenting shows grant/config impact |
| ORG-02 / 5 | ORG-01 | Location hierarchy/address/timezone and team definitions/membership UI | Physical locations and teams are distinct; off-campus facility works; team membership alone grants no access |
| ORG-03 / 5 | ORG-01, CUR-01 | Multi-curriculum assignment to schools/years/cohorts and effective-date UI | Two curricula coexist in one school; published versions stable; transfer/grade rules selected explicitly |
| CTX-01 / 8 | TEN-03, scoped IAM-03 slice | Authorized tenant/org/school/campus/team context selector and aggregate-scope labels | Forged context denied; create selects one owner; dirty switch cancel retains original context; failed switch never relabels old data |
| CTX-02 / 8 | CTX-01 | Context epoch/cancellation, full cache/draft/blob/export cleanup across web/Wails/mobile | Late reads/writes/401s cannot affect new context; two tabs, mobile resume and revoked grant tests pass |
| CFG-02 / 8 | ORG-01, CFG-01 | Schema-defined inheritance and locked policy levels; effective value/source, override/reset, publish diff UI | Personal settings cannot weaken policies; reparenting previews changes; concurrent publish conflicts preserve draft |
| UI-07 / 8 | CFG-02 | Server-backed shared workspace presets and personal overrides with bounded versioned schemas | Navigation/order/default views/column layout/sections synchronize where supported; reset-to-inherited differs from reset defaults |
| UI-08 / 8 | UI-07 | Keyboard/touch layout customization, view/column/safe-filter presets, import/export validation | Unknown/unsafe config rejected; hidden required errors revealed; imported layout never exposes denied module/action or row values |
| LNG-01 / 5 | CFG-02 | Locale registry, UI/content/instruction/output language separation, formatting/zone contracts | EN/FR complete; locale fallback explicit; non-supported locale cannot break login; stored values locale-independent |
| LNG-02 / 8 | LNG-01 | Shared multilingual field editor, language completeness/fallback and immutable output snapshots | Editing one language preserves other drafts; Unicode/import/export round-trip; issued transcript wording does not change later |
| MOV-01 / 8 | TEN-04 | Versioned per-module tenant transfer manifests and coverage check | Every tenant table/blob/event included or justified; new unregistered table fails architecture gate |
| MOV-02 / 8 | MOV-01 | Tenant database transaction gate/fence and worker/upload/export participation | Race long transaction vs fence, stale process and delayed delivery; no split writer; bounded maintenance mode visible to clients |
| MOV-03 / 8 | MOV-02 | Resumable same-engine move orchestrator with preflight/copy/validate/CAS/activate/observe | Shared↔dedicated and pool↔pool moves preserve IDs/hashes; crash at each state resumes; other tenants unaffected |
| MOV-04 / 8 | MOV-03 | Typed SQLite↔PostgreSQL converters and capacity/schema checks | Money/decimal/time/Unicode/nullable data preserve meaning in both directions; no raw dump assumption |
| MOV-05 / 8 | MOV-03 | Operator placement inventory, preflight, maintenance preview, start/status/history/recovery UI | Authorized operators only; safe redacted status; double start harmless; restart reconnects job; cancel/recovery match backend phase |
| MOV-06 / 8 | MOV-04..05, OPS-06 | Move recovery/restore/rollback drills and runbooks | After target writes, rollback is verified reverse move; old control restore fails epoch checks; evidence for all supported directions |

**Dependency ruling:** deliver minimal scoped-role/organization provisioning contracts together with TEN-03 to break CTX/IAM onboarding dependencies; fuller role editor follows in S03. CFG-01 typed settings schema precedes CFG-02; no untyped mutable configuration blob. CUR-01 precedes ORG-03; ORG-03 final acceptance therefore completes with S04. Enabling stories are scheduled explicitly, not replaced with mocks.

## Full product backlog: domain and UI acceptance

All domain stories below include unit rules, SQLite/PostgreSQL repository tests, API permission/tenant tests and the named end-to-end journey. API resources are under `/api/v1`; create returns 201, invalid input 422, absent/cross-tenant resources 404, denied operations 403 and version conflicts 409. List endpoints have bounded pagination and stable ordering. Financial commands require idempotency keys; updates carry versions. State-changing actions create an auditable event atomically.

| Story / pts | Module / dependencies | Business deliverable | Complete frontend deliverable | Acceptance journey |
|---|---|---|---|---|
| TEN-01 / 8 | tenancy / TEN-02..04 | Tenant/group metadata, active/suspended lifecycle, explicit authorized context | Group profile, scope-aware onboarding, lifecycle impact preview and safe placement status | Suspend tenant blocks its work; school suspension affects only that school; unrelated tenant remains available |
| IAM-01 / 8 | auth / TEN-01 | Secure account lifecycle, password change/reset tokens, session inventory/revocation and login throttling | Sign-in, password recovery/change, session list with revoke controls | Invalid/expired/reused reset token rejected; password change revokes other sessions |
| IAM-02 / 5 | auth / IAM-01 | Member list/create/update/disable, invitations via delivery port, last-admin protection | Searchable member directory, add member/invite, profile, deactivate confirmation | Admin creates registrar; registrar denied role changes; final admin cannot be removed |
| IAM-03 / 8 | auth / IAM-02 | Permission catalog and role grants, class/self/family scopes, server-enforced policy | Role editor, effective-access preview, assignment workflow | Teacher cannot read unassigned class; guardian cannot read unrelated child |
| CFG-01 / 5 | school_settings / TEN-01 | School locale/timezone/currency/calendar defaults and versioned settings | Shared settings sections, clear validation, save/reset and conflict recovery | Invalid timezone rejected; valid setting changes formatting across clients |
| SUB-01 / 8 | subscriptions / TEN-01 | Plan/entitlements and explicit lifecycle with provider port; no silent loss of data | Plan/usage/renewal status and administrator entitlement controls | Disabled feature is denied server-side while existing authorized export remains available |
| ACD-01 / 5 | academics / CFG-01 | Academic years, ordered nonoverlapping terms, open/closed lifecycle | Year/term list, editor, calendar summary and close confirmation | Overlap rejected; closed term blocks new grades and attendance |
| CUR-01 / 5 | curriculum / ACD-01 | Subject codes, grade-level curricula, versioned assessment weights | Subject catalog, curriculum detail and weight editor | Duplicate code rejected; publication requires weights total exactly 100% |
| CLS-01 / 8 | classes / CUR-01, IAM-03 | Classes/sections/capacity, teaching assignments and room links | Class directory, roster, staff assignment and capacity indicators | Over-capacity enrollment denied; teacher sees assigned roster only |
| ADM-01 / 8 | admissions / ACD-01 | Applicant data/documents, draft/submitted/review/accepted/rejected transitions and conversion | Intake form, review board/list, applicant detail and decision dialog | Admission conversion is idempotent and creates one student with audit trail |
| STU-01 / 8 | students / ADM-01 | Student profile/identifier/contact fields, archive reason, search and controlled import | Full student list/detail with profile, enrollment, guardians and history tabs; import preview | Duplicate identifier rejected; import reports row errors before atomic accepted commit |
| GUA-01 / 5 | guardians / STU-01 | Guardian contacts and verified relationships with permission/consent flags | Family directory, relationship editor and linked children view | Family relationship removal immediately removes child access |
| ENR-01 / 8 | enrollment / CLS-01, STU-01 | Placement, section assignment, transfer, promotion and withdrawal history | Enrollment roster, placement wizard, transfer and promotion preview | Promotion retries do not duplicate records; historical placements preserved |
| ATT-01 / 8 | attendance / ENR-01 | Per-class/date/session marks, corrections, absence reason and locked periods | Date/class roster, keyboard mark entry, review/submit and correction history | Duplicate submission updates one mark; unassigned teacher denied; late correction audited |
| TTM-01 / 8 | timetables / CLS-01, STF-01 | Weekly slots, effective dates, room/teacher/class conflict rules and publication | Weekly calendar plus accessible list alternative, slot editor and conflict panel | Overlapping teacher/room slots rejected; unpublished changes hidden from learners |
| LRN-01 / 8 | learning / CLS-01, DOC-01 | Learning resources, assignment deadlines, submissions and feedback | Teacher assignment editor, student task list/submission and feedback detail | Student submits own work only; late policy explicit; withdrawn student denied |
| ASM-01 / 8 | assessments / CUR-01, CLS-01 | Assessment definitions, max marks, weight and draft/published lifecycle | Assessment list/detail, setup form, publish checklist | Invalid weighting or zero maximum rejected; draft not visible to students |
| GRD-01 / 8 | gradebook / ASM-01 | Exact decimal grade calculation, missing/excused distinction and moderation locks | Accessible marks grid with non-grid alternative, totals and conflict resolution | Same inputs produce deterministic totals; failed batch preserves existing marks |
| EXM-01 / 8 | examinations / TTM-01, ASM-01 | Exam sessions, candidate registration, seating and invigilation conflicts | Session scheduler, candidate/room allocation and printable roster | Double-booked candidate/invigilator rejected; candidate lists tenant-scoped |
| TRN-01 / 8 | transcripts / GRD-01, EXM-01 | Versioned publishable report snapshots, approval and revocation | Report preview, moderation queue, publish dialog and accessible print/download | Grade edits after publication do not mutate issued transcript; revocation visible |
| DSC-01 / 8 | discipline / STU-01, IAM-03 | Restricted cases, incidents, actions, follow-up and visibility scopes | Case list/detail with timeline, action editor and restricted disclosure | Unauthorized staff receive no case details in list/search/export/audit payloads |
| SCH-01 / 5 | scholarships / STU-01, WFL-01 | Award rules, applications, capped allocations and approval | Award catalog, application review and remaining-allocation view | Over-allocation rejected under concurrency; revoked award adjusts future charges only |
| FEE-01 / 5 | fees / ENR-01 | Currency-specific fee schedules, effective dates, installments and concessions | Schedule editor, installment table and student charge preview | Negative amounts rejected; approved schedule change does not rewrite issued invoices |
| BIL-01 / 8 | billing / FEE-01, SCH-01 | Draft/issued/void invoices, immutable lines, credit notes and aging | Invoice list/detail/editor, issue confirmation, credit note and statement print | Same issuance key produces one invoice; credits cannot exceed permitted balance |
| PAY-01 / 8 | payments / BIL-01 | Provider/manual payment ports, verified webhooks, allocations, refunds and reconciliation | Receipt entry, payment detail, unmatched-payment queue and refund approval | Duplicate/out-of-order webhooks preserve correct balance; refund never exceeds payment |
| ACC-01 / 8 | accounting / PAY-01 | Chart of accounts, balanced journals, posting/reversal and fiscal close | Account ledger, journal editor/review/post, trial balance and close checklist | Unbalanced journal rejected atomically; closed period requires controlled reversal path |
| STF-01 / 5 | staff / IAM-02 | Staff identifier/profile, employment dates and teaching eligibility | Staff directory/detail, profile editor and assignment summary | Duplicate staff code denied; inactive staff cannot receive new assignments |
| HR-01 / 8 | hr / STF-01, WFL-01 | Contracts, leave balances/requests, approval and restricted employment files | HR profile, leave calendar/request/review and document access | Leave approval cannot produce negative balance; medical/private files restricted |
| PRL-01 / 8 | payroll / HR-01, ACC-01 | Configurable earning/deduction rules, draft/review/approve/export runs and immutable payslips | Run wizard, exceptions, payroll review, approval and payslip access | Re-run is idempotent; approval locks lines; each staff member sees own payslip only |
| PRC-01 / 8 | procurement / ACC-01, WFL-01 | Supplier, requisition, approval, purchase order and receiving states | Supplier list, requisition editor, approval inbox and receipt form | Cannot receive beyond ordered quantity without approved amendment |
| INV-01 / 8 | inventory / PRC-01 | Items/locations, receipts/issues/transfers and immutable stock ledger | Item list/detail, stock movement forms and stocktake reconciliation | Concurrent issues cannot drive stock below policy floor; transfer balances both locations |
| FAC-01 / 5 | facilities / INV-01 | Rooms/assets, maintenance tickets, assignments and booking conflicts | Space/asset directory, booking calendar/list and maintenance board | Overlapping exclusive booking rejected; maintenance closure records resolution |
| LIB-01 / 8 | library / STU-01, STF-01 | Bibliographic records/copies, loan/reservation/return and configurable overdue rules | Catalog/copy detail, checkout/return desk, reservations and overdue list | Same copy cannot have two active loans; return retry is harmless |
| TRP-01 / 8 | transport / STU-01, STF-01 | Routes/stops, vehicles/capacity, rider assignments and trip boarding | Route editor, rider roster, vehicle assignment and mobile boarding checklist | Capacity and duplicate assignment enforced; boarding restricted to trip staff |
| NTF-01 / 8 | notifications / IAM-02 | Delivery preferences, templates, queued send/retry/dead-letter and provider ports | Notification inbox, preference settings and operator delivery diagnostics | Retry delivers at most once per provider key; unsubscribed category suppressed |
| MSG-01 / 8 | messaging / GUA-01, IAM-03 | Scoped conversations/participants/messages, read markers and moderation | Inbox/thread/composer, recipient picker and delivery state | Removed participant cannot fetch later messages; no arbitrary school-wide recipient expansion |
| WFL-01 / 8 | workflows / IAM-03 | Versioned approval definitions, instances, assignments, approve/reject/cancel and audit | Workflow definition editor, assigned-task inbox and instance timeline | Two approvers racing cannot advance the same step twice; self-approval policy enforced |
| DOC-01 / 8 | documents / IAM-03 | File metadata, ownership/link scopes, list/search, retention, safe deletion and scan/quarantine port | Document library, upload with progress, preview/download, details and delete confirmation | Quarantined files cannot download; filename/type/size errors clear; unrelated role denied |
| AUD-01 / 5 | audit / IAM-03 | Append-only audit query with actor/action/resource/date filters and redaction | Audit table/detail, safe filters and authorized export | Tenant filter mandatory; sensitive secrets never enter audit payload; no UI edit/delete |
| POR-01 / 8 | portal / GUA-01, ATT-01, TRN-01, BIL-01 | Role-scoped student/family summaries, schedules, statements and consent actions | Family child switcher, student home, attendance/results/fees/documents pages | Family sees only linked children; unpublished results and staff-only cases absent |
| RPT-01 / 8 | reporting / domain queries | Versioned report definitions, parameter validation, async exports and expiring downloads | Report catalog, parameter form, preview and export jobs/history | Export permissions rechecked at execution/download; CSV formula injection neutralized |
| ANL-01 / 8 | analytics / RPT-01 | Reproducible aggregates, refresh timestamps and permitted drill-down | Real KPI summaries, accessible charts/tables, filters and explanatory empty states | Dashboard total agrees with source report; filters/timezone applied consistently |

**Dependency rulings:** STF-01 and the core WFL-01 engine are enabling stories pulled forward into S04/S03 respectively. DOC-01 metadata/authorization is pulled forward into S05; advanced retention/scan administration completes in S16. TTM-01 and LRN-01 cannot start until these prerequisites pass. The main sprint table groups final domain acceptance rather than prohibiting prerequisite slices. No circular dependency is satisfied by a placeholder.

## Full frontend backlog

| Story / pts | Deliverable and owned files | Acceptance |
|---|---|---|
| UI-01 / 5 | Shared responsive shell/navigation, route metadata and capability-aware actions (`modules/core/resources/js/layouts`, `app/router.ts`, shared client contracts) | Mobile navigation closes/restores focus; active page/breadcrumb correct; revoked permission removes actions; no cross-user content flash |
| UI-02 / 8 | Shared list/detail/editor/calendar/board/roster/ledger/report/settings templates (`components`, `features`) | Each template has a real consuming domain page, consistent spacing and all asynchronous states; no unused template catalog masquerading as pages |
| UI-03 / 5 | Workspace preferences: theme, accent, density, width, pins/order, selected views/columns (`features/preferences`, `app/preferences.ts`) | Changes apply across pages and Wails; reset works; school/user isolation; denied storage handled gracefully |
| UI-04 / 8 | Durable navigation and recovery: URL-safe filters, unsaved-change dialog, error boundary, session/offline/conflict feedback | Browser back/forward preserves safe filters, cancellation retains drafts, unknown write result never says success, no native confirm dialog for app navigation |
| UI-05 / 5 | English/French content audit and accessibility baseline, visual references (`DESIGN.md`, `UX-CONTRACT.md`, UI tests) | Labels, validation, status and formatting localized; keyboard/320px/200%-zoom checks; accessible dialogs and error summaries |
| UI-06 / 8 | Domain screen matrix and real-browser regression (`tests/e2e`, `docs/product/frontend-matrix.md`) | Every domain story names and implements list/detail/action screens; empty/success/failure/denied states exercised; desktop and narrow captures reviewed |

## Cross-cutting production completeness

| Story | Deliverable | Acceptance / placement |
|---|---|---|
| IAM-04 | Administrator MFA enrollment/recovery and optional OIDC provider port | Recovery codes single-use/hashed, enrollment confirmation, bounded step-up for grants and moves; S03/S20 |
| IAM-05 | Explicit account linking and scoped delegation/revocation contract | Same email never links automatically; time-limited support grant and removal audited; in-flight revocation semantics tested; S03 |
| DATA-01 | Governed person linkage, consent, retention and deletion/export requests | School-scoped profiles stay private; legal hold blocks purge; authorized export/redaction and recovery consequences explicit; S05/S16 |
| OPS-10 | Per-tenant quotas, rate limits and workload fairness | Noisy tenant cannot starve peers' API/jobs/uploads/exports; overload gives bounded actionable retry; S18 |
| OPS-11 | Tenant-specific readiness and capability/version compatibility | One moving/suspended tenant does not unnecessarily remove healthy peers; no incompatible schema or stale-route fallback; S18 |
| REL-07 | Threat model, dependency/supply-chain and privacy review | Tenant-scope leaks, privilege delegation, IDOR, SSRF, CSV/XSS/file threats, sensitive log redaction and secrets rotation tested/documented; S20 |

**Release scope rule:** these are required release stories even though the main sprint table groups them by epic. Provider adapters expose deterministic sandbox/local providers for development; live credentials and operator-specific policies are external gates. SSO support ships only for explicitly tested providers. No legal/payroll compliance is inferred from configurable calculation rules.

## Foundation, clients and release backlog

| Story / pts | Deliverable | Acceptance / evidence |
|---|---|---|
| OPS-01 / 5 | Native-first Make targets, supervised dev stack, optional Docker targets, setup diagnostics | Native SQLite migrate/seed/API/worker/UI works without Docker executable; failure shuts down children and exits nonzero |
| OPS-02 / 3 | Product naming correction | The `EduPilotOS` provider boots and each client identifies itself as EduPilot OS |
| OPS-03 / 5 | Execute all existing suites and repair real defects | Root Go, web/mobile, gateway and available browser tests executed; failures fixed with regressions; unavailable services explicitly recorded |
| OPS-04 / 5 | Automated native CI and reproducible evidence | Pull requests run tests/builds without Docker requirement; PostgreSQL integration job and optional container smoke separate; secret-free artifacts |
| OPS-05 / 8 | Native build/package and install/uninstall/service units | Built server serves production UI; API/worker/migrator separately runnable; least-privilege system service and private config; startup/readiness tests |
| OPS-06 / 8 | Backup/restore including control, tenant DBs, files and keys | Restore disposable installation, compare record/file hashes and placement markers; documented RPO/RTO from measured drill |
| OPS-07 / 5 | Structured/redacted logs, request correlation, readiness, metrics and alerts | Worker lag/failures, DB/readiness and API errors observable; credentials and student payloads absent from logs |
| OPS-08 / 8 | Upgrade/rollback and failure recovery | Upgrade prior snapshot, crash between migration/pool receipts, recover without duplicate effects; incompatible downgrade fails clearly |
| OPS-09 / 5 | Dependency/license and security maintenance | Lockfiles, vulnerability inventory, SBOM, pinned release checks and responsible disclosure route; critical/high findings resolved before release |
| CLI-01 / 5 | Wails complete shared UI and secure gateway packaging | Same workflows as web, configured origin restrictions, binary uploads, native window smoke; Windows/macOS/Linux release matrix |
| CLI-02 / 8 | Expo staff daily flows | Login/recovery, class roster, attendance, assignments, messages, files and notifications work with live API; offline/expiry behavior tested |
| CLI-03 / 8 | Expo family/student flows | Linked child switcher, timetable, attendance, published grades, fee statements and communication; device-secure session handling |
| CLI-04 / 8 | Device/browser release QA | Chromium/Firefox/WebKit and mobile size journeys; actual Android/iOS/desktop smoke, deep links, upgrades and signed packages |
| REL-01 / 8 | Adversarial security/tenant/role review | Auth, sessions, file access, exports, webhook signatures, rate limits and role matrix negative tests; no unresolved critical/high issue |
| REL-02 / 5 | Accessibility and visual review | WCAG 2.2 AA review, automated checks and keyboard/manual screen-reader pass on critical journeys; EN/FR and all themes |
| REL-03 / 8 | Performance/capacity and resilience | Agreed deployment-size fixture and load profile; record p50/p95/errors, DB contention and worker catch-up; budgets approved from measurements |
| REL-04 / 5 | School UAT and policy approval | Representative admin/teacher/finance/family journeys, grading/currency/payroll/retention policies signed off by school operator |
| REL-05 / 5 | Release operations and integration readiness | Hosting/domain/TLS, live provider accounts, backups, incident/rollback owners and signing identities configured and verified |
| REL-06 / 5 | Versioned release candidate | Changelog, migration notes, checksums/SBOM/signatures, release checklist and linked evidence; release only when mandatory gates green |

## Task 1: Publish scope and Scrum backlog

**Files:** `docs/product/production-scope.md`, this plan, `docs/product/frontend-matrix.md`, `docs/release/status.md`.
**Interfaces:** Produces stable story IDs and acceptance criteria consumed by all implementation increments.

- [x] Check all planned workflows map to a story and full UI row; no backend-only completion definition.
- [x] Record foundation baseline and existing authorization, native default and naming rulings.
- [x] Commit and push plan before product implementation. Expected: remote commit/tree contains these documents.

## Task 2: Native operation and baseline qualification (S01)

**Files:** `Makefile`, `scripts/dev.py`, `scripts/seed-admin.sh`, `scripts/ci-e2e.sh`, `scripts/tests/test_dev.py`, `README.md`, `docs/local-deployment.md`, `app/providers/edupilot_os.go`, `bootstrap/providers.go`, `.github/workflows/verification.yml`, `docs/verification.md`.
**Interfaces:** `python3 scripts/dev.py` runs migration then supervises API, worker and Vite; `make up` runs it in foreground; `make migrate`, `make seed-api` are native. `make docker-up|docker-down|docker-logs|docker-ps|docker-migrate|docker-seed` explicitly select containers. `--check` validates prerequisites without modifying data.

- [ ] Execute baseline suites separately, write exit status and skip names to release evidence.
- [ ] Write native launcher tests for missing dependency, migration failure, child failure and shutdown; run RED.
- [ ] Implement native supervisor with process-group cleanup and no shell interpolation of credentials. Keep child diagnostics and return failure codes.
- [ ] Verify the application provider resolves installed module services through the Goravel container.
- [ ] Make native docs/commands primary; retain explicit Docker commands and update optional container workflow references.
- [ ] Run native launcher tests, root Go tests, web/mobile/gateway tests, web build, native install smoke and real browser tests where supported. Expected: green suites; unavailable checks named with reasons.
- [ ] Commit/push coherent checkpoints and update status with verified evidence.

## Task 3: Deliver group tenancy and full frontend foundation (S02)

**Files:** `modules/core/access`, `modules/core/database`, `modules/auth`, `modules/tenancy`, `database/edumigrations`, `clients/web/src/app`, `modules/core/resources/js/layouts`, `modules/core/resources/js/components`, `modules/core/resources/js/features/preferences`, `clients/web/src/tests`, `tests/e2e/specs`, design documents.
**Interfaces:** Extend existing `Preferences`, `readRoute` and shared primitives; keep API DTOs in `packages/api-client`. Use one shared confirmation/navigation controller for dirty editors, not per-page browser dialogs. Route metadata maps existing module keys to UI capability/status; it does not grant permissions.

- [ ] Refine TEN-02..04, ORG/CTX/CFG/LNG enabling stories from the tenancy specification before broad UI/domain changes. Implement scope mapping/migration/authorization first with upgrade and sibling-scope isolation tests.
- [ ] Add tests for safe route state, malformed URLs, preference reset/reorder, mobile menu focus, dirty-navigation cancel/confirm, context-switch races, inherited-setting locks and multilingual field preservation; watch new expectations fail.
- [ ] Implement reusable shell/route/dialog/preference behavior and adapt the Auth and Tenancy pages.
- [ ] Add full-flow browser cases for each implemented screen with desktop/narrow layouts, keyboard, French/dark theme, empty/error/offline and conflict states.
- [ ] Inspect real screenshots and fix overflow, hierarchy, contrast and focus defects. Record exact coverage in the frontend matrix.
- [ ] Run component/API tests, browser suite and build. Request independent frontend/code review, resolve important findings, commit/push.

## Task 4: Refine and deliver S03–S17 in dependency order

**Files:** Each owner `modules/<key>/{app/models,app/services,app/http/controllers,routes,database/migrations,providers,resources}` following its existing ownership layout, `app/schoolapi`, shared DTO/OpenAPI and `modules/<key>/resources/js` as appropriate; module/unit/feature/e2e tests.
**Interfaces:** Reuse existing `access.Principal`, tenant catalog, `outbox.Append`, Core contracts composed at bootstrap. Introduce named public read/command contracts at each module boundary. Never query another module's private tables directly.

For each story in the domain table:

- [ ] Refine the upcoming story into a file-level subplan containing concrete DTO fields, states, role grants, migrations/indexes, API routes/errors and screen flow. Split any 13-point or multi-actor story into demonstrable vertical slices.
- [ ] Write domain/permission/concurrency tests first; run RED; implement model rules and service contracts; run GREEN.
- [ ] Write real SQLite/PostgreSQL persistence/API/contract tests; implement module services, atomic events and Blueprint migrations; prove negative tenant cases.
- [ ] Write UI interaction and end-to-end tests; implement full page flow from shared templates and real APIs; prove denied/empty/error/dirty/conflict behavior.
- [ ] Update OpenAPI, frontend matrix, role matrix and module documentation; run the affected and cross-module suites, review, commit and push.
- [ ] Mark the story Verified only with recorded acceptance evidence. Repeat until all planned domain stories meet the definition of done.

## Task 5: Deliver S18–S20 release qualification

- [ ] Refine each operations/client/release story into concrete artifacts and executable checks using its backlog acceptance criteria.
- [ ] Run both database modes, migration/restore/upgrade drills, role/tenant matrix, browser/device matrix and integration sandboxes.
- [ ] Perform independent whole-branch code/security/frontend review and remediate critical/high/important findings with regression tests.
- [ ] Publish a release evidence manifest with commit SHA, environment, command, timestamp, result, skip reason and linked logs. Remove test credentials from artifacts.
- [ ] Keep external prerequisites blocked until actually supplied/verified. Document the exact operator action and affected release gate.
- [ ] Package/tag only a qualified version with release notes, checksums, supported-platform matrix and operator runbook. Push implementation and documentation; deploy only under deployment authorization.

## Release gate matrix

| Gate | Required evidence | Initial state |
|---|---|---|
| Scope | Every listed module workflow and frontend row accepted | Open |
| Automated tests | Unit, feature, contract, integration and e2e green on release SHA | Open |
| PostgreSQL/SQLite | Isolation, concurrent writes, migrations, backups and restore | Open |
| Group tenancy | Multiple organizations/schools/campuses/teams/curricula in one tenant; scoped grants/configuration and verified whole-tenant relocation | Open |
| Tenant mobility | Shared/isolated and cross-engine moves; fence/epoch/crash/reverse-move/restore drills | Open |
| Full UI | Shared components, complete page flows, context isolation, inherited customization, multilingual fields and responsive/keyboard review | Open |
| Security | Access/tenancy/file/export/webhook/session review and dependency audit | Open |
| Operations | Native clean install, upgrade, recovery, monitoring and rollback drills | Open |
| Native clients | Signed packages and actual supported device/window smoke evidence | External setup required |
| Providers | Sandbox tests plus operator-configured production account readiness | External setup required |
| School policy/UAT | Configured policy validation and representative operator acceptance | External input required |
| Deployment | Hosting, domain/TLS, secrets, backup target and named operator | External input required |

No gate is waived by a skipped test or an implementation estimate. The final status must distinguish implemented, verified, blocked and released.
