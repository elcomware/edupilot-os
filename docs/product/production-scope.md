# EduPilot OS production scope

Product name: **EduPilot OS**. Repository and technical identifier: **edupilot-os**.

## Outcome

A school can install EduPilot OS natively, configure its institution and academic policies, invite staff, operate the complete student lifecycle, teach and assess, collect and reconcile fees, manage staff and resources, and publish appropriately scoped information to families. Web, desktop and mobile use the same authenticated API. Every shipped capability has a usable interface, persisted business behavior and verified authorization. A module directory or a general work list does not complete a business module.

The starting point is commit `ae7a791cdbbdc1bf76be7f1a166e9a226c3da177`: 42 module registrations; working student registration, enrollment placements, document storage, sessions and per-module work lists; shared React web/Wails UI and Expo client. The original 20 iterations were a starter phase, not a production release.

## Binding constraints

- Preserve the standard Goravel 1.18 application tree and hexagonal module boundaries. Domain/application code has no HTTP, SQL or Goravel dependency.
- Use **EduPilot OS** for product copy and **edupilot-os** for repository/service identifiers. School remains the legitimate domain term for a tenant.
- Native installation and operation are the documented default. Docker is optional and has explicit `docker-*` commands.
- Support SQLite for a single-process modest deployment and PostgreSQL for concurrent deployment. Keep tenant placement explicit, including dedicated databases.
- Every read, write, background job, export and file operation is authorized using a verified principal and tenant. Never trust a client-supplied tenant switch.
- Full UI is part of each story's delivery: useful page flows, real APIs, responsive layout, professional typography, keyboard operation and meaningful states. No dummy statistics or controls with no effect.
- Share tokens, shell, page layouts, data tables, forms, dialogs and state components. Domain pages supply typed data and actions; avoid copying whole pages.
- Customize language, theme, accent, density, width, navigation and permitted workspace views per user and school. Preferences must never grant permissions.
- English and French are supported throughout UI, validation and printed output. Dates use the configured school timezone; money uses integer minor units and explicit currency.
- Web and Wails share the complete administrative interface. Expo supplies role-appropriate daily workflows; the release matrix must state which workflows require the administrative web/desktop client.
- Write unit, feature/integration, contract and end-to-end tests. Run available tests. Record an unavailable test with its reason and unblock requirement. A skipped gate is not a passing gate.
- Push the plan before implementation. Push coherent, reviewed implementation checkpoints to the authorized repository. No production deployment, real customer data import or external messages are implied.

## Professional frontend contract

Continue the existing restrained school-register design in `DESIGN.md`. A consistent navigation rail, school context, breadcrumb, heading and primary action frame each workspace. Add view templates for record list/detail/editor, calendar, board, roster, financial ledger, report and settings. Choose the template to suit the task; do not force all business screens into a generic title/status editor.

Every route specifies its intended role, permissions, URL state, empty/loading/error/success/forbidden/offline/conflict states, mobile behavior and test. Tables support keyboard-accessible actions, named column controls, stable sort and explicit page size. Filters and pagination are shareable in the URL when safe. Sensitive drafts and tokens are never put in URLs or cosmetic preferences. Long names, long French labels, 320px width and 200% zoom must remain usable. Status uses text as well as color. Confirm destructive actions in a shared accessible dialog, preserve drafts on errors, return focus predictably and announce asynchronous results.

Navigation is capability-aware. Students/families see only their own or linked records; teachers see assigned classes; administrative roles see explicitly granted modules. Unimplemented domain features must be labeled as unavailable in the delivery matrix, never represented as completed by a workspace note.

## Policy assumptions and release prerequisites

The implementation will make school-specific grading scales, attendance categories, terms, fee schedules, currencies, approval thresholds and retention periods configurable. It must not assume one country's tax, payroll, safeguarding or reporting rules. Use explicit draft/approve/post/publish transitions where actions have financial or academic consequences. Posted records are reversed, not silently rewritten.

External email/SMS/payment providers remain behind adapters with deterministic local test providers. A live release requires configured production provider credentials, verified sender/payment accounts, approved school policies, a hosting target, domain/TLS, backup destination and platform signing identities. These are tracked as external release gates; missing values do not justify skipping implementation of contracts, sandboxes or error paths.

## Definition of production release ready

All committed release-scope stories meet their backend **and UI** acceptance criteria; required automated suites pass on SQLite and PostgreSQL; migration/restore and tenant isolation drills pass; security and accessibility reviews have no unresolved critical/high issues; operators have a tested native install, monitoring, backup, recovery and rollback procedure; signed native packages and device smoke checks exist for supported platforms; external integrations pass provider sandbox and configured live readiness checks; an authorized operator approves the deployment-specific release checklist. No claim of production readiness is made from compilation alone.
