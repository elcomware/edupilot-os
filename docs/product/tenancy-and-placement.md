# EduPilot OS tenancy, organization and placement specification

This specification extends the production scope. A single group tenant may contain multiple organizations, schools, campuses, locations, teams, curricula and languages, and the entire tenant can move between shared or isolated database placements.

## Identity and ownership model

| Entity | Meaning / parent | Required invariants |
|---|---|---|
| Tenant | Customer/security/data-placement boundary; may be a school group | Stable opaque ID, unique public slug, lifecycle and one authoritative active data placement; no implicit parent-tenant access |
| Organization | Legal or administrative entity within one tenant; optional parent organization | Parent is in same tenant; acyclic tree; legal currency/accounting policy explicit; reparenting audited and reauthorizes inherited grants |
| School | Academic operating institution within one organization | Stable ID distinct from tenant; own calendar, identifiers, policies, languages and one or more campuses; no automatic access from another school |
| Campus | School operating site | Exactly one school and tenant; campus code unique in school; active dates and timezone policy explicit |
| Location | Physical site/building/floor/room or off-campus facility | Organization-owned; optional school/campus owner; parent chain cannot cross tenant or contradict ownership; address/country/timezone stored explicitly |
| Team | A named group for work/communication/authorization assignment | Tenant-owned with an explicit scope; membership is separate from grants; cross-school teams need explicit grants for each allowed school; membership alone grants no records |
| Curriculum | Tenant-owned, versioned educational framework | Assigned explicitly to school/year/cohort; multiple simultaneous curricula allowed; published version immutable; subject equivalence and transfer rules explicit |
| User account | Authentication identity | Stable subject identity; tenant memberships explicit; never merge existing accounts merely because email strings match |
| Membership / grant | Subject or team + action/role + scope | Explicit tenant and bounded scope, lifecycle/expiry and version; deny by default; no role-string shortcut to all tenants |
| Workspace | A user/task arrangement of views and actions | Contains only currently authorized context/data; layout preferences never change grants or scope |

Identifiers do not encode hierarchy, location or physical database. Moving a tenant preserves every identifier. A school transfer between organizations or tenants is a separate domain operation with separate policy/data migration; it is not a database placement move. Independent tenants stay independent even when the same operator owns them.

Tenant is the first-release placement unit. All organizations and schools in that tenant move together. Splitting one tenant's schools over multiple active databases would require distributed module transactions and is outside this single-placement design; no UI suggests it is already supported. A dedicated database isolates SQL data, not automatically processes, object storage, queues, identity services or encryption keys. Record each isolation dimension explicitly.

## Student/person and revocation rulings

Maintain a tenant-level person identity only through an explicit verified linkage. School enrollment/profile/disclosure remains school-scoped; matching a name, email or birth date never merges people or reveals sibling-school records. Default new school records are independent until an authorized linkage workflow establishes a shared person reference. The UI displays the distinction and records linkage/reversal history.

Membership/grant revocation prevents subsequent authorization checks immediately. An already-authorized in-flight command can complete until its bounded transaction ends unless the operation uses a scope fence. Tenant suspension and placement moves require strict drain/fence behavior. High-risk access removal uses a bounded scoped fence; do not claim instantaneous cross-database revocation from a request-entry check alone. Document the measured maximum lease/transaction duration, test races and fail closed if authority cannot be read. Relationship checks in the tenant database are made inside the business transaction.

## Control plane and tenant data

| Control database | Tenant data pool |
|---|---|
| Tenant identity/slug/lifecycle; database pool registry; placement epochs; move journal and operator audit; verified authentication identities and session grants | Organization/school/campus/location records; academic and business records; team definitions and scoped policy data where appropriate; module workspaces; tenant outbox/inbox; documents metadata; tenant audit |
| Minimal context index needed to resolve and authorize a session; versioned membership/grant references | Authoritative hierarchy and scope relationships used by business operations; changes invalidate the control index before access can expand |
| Secret references and pool identity, never raw connection passwords in HTTP responses | Explicit tenant_id on every tenant-owned row, including in dedicated databases |

For the initial migration, keeping all authorization grants in the control database avoids an eventual-consistency authorization decision. Hierarchy changes that affect grants are orchestrated with deny-first behavior: invalidate affected grants/context versions before changing the tenant hierarchy, then rebuild/re-enable validated grants. A later optimized hierarchy index cannot become an authority that silently grants access while stale.

Separate named shared data pools from the control connection in the placement registry. The default developer deployment may use one SQLite database for both, but neither APIs nor operational moves assume that physical coincidence. Pool registry entries have opaque pool ID, driver, placement class (shared/dedicated), region, secret reference and supported schema version. Dedicated pools permit one tenant; shared pools permit many. Configuration registers physical connections; only the durable placement record selects a tenant's active pool.

Business data uses composite tenant-aware keys and foreign keys, and organization/school/campus IDs on rows at those ownership levels. A nullable scope is not interpreted as wildcard access. Group-wide objects have an explicit tenant-level ownership kind. Cross-school reports call public authorized query contracts; they do not remove tenant/school predicates.

## Verified request context and permissions

The target verified context contains subject ID, tenant ID, selected organization/school/campus/team IDs where applicable, effective locale, grant version and placement epoch. A selected context is a request to narrow authorized work, not a permission grant. The API loads and validates the full parent chain and role scope server-side on each command. Client headers/query parameters alone never authorize scope.

Every public module port takes a verified context and an explicit command/query. Permission decisions consider action, tenant, resource ownership, school/campus/class/team assignment, self/family relationship and record lifecycle. Tenant administrators are distinct from platform operators. Support access requires a time-bounded explicit delegation, visible reason and audit trail; there is no hidden universal bypass.

Cross-context switching is explicit: list authorized choices → validate chosen context → issue updated session/context → cancel requests, clear old query caches and sensitive drafts, apply the new authorized catalog → render. Unsaved work is resolved first. A late response from an old context is discarded. Tabs and devices cannot infer each other's active tenant. Download/export jobs retain the original verified scope and recheck current permissions at execution and retrieval.

Role acceptance fixtures include: platform operator with no tenant data grant; group administrator; organization administrator; school administrator; campus manager; class-assigned teacher; finance staff; HR staff; librarian; driver; student; guardian; time-limited support delegate. Test allowed and denied operations for siblings under the same tenant as well as entirely different tenants.

## Hierarchical configuration and language

Configuration precedence is explicit and per setting: product default → tenant → organization → school → campus → personal preference, where the key permits an override. Each setting schema names allowed levels, validation and whether a lower level can override it. Security, retention and mandatory school policy cannot be weakened by personal preferences. The UI shows effective value, source, override/reset-to-inherited and affected descendants before a broad change.

Ship English and French UI catalogs first, with a locale registry that can add additional supported catalogs and declare text direction. UI language, content language, school teaching language and curriculum language are separate. User-entered translatable fields carry language-tagged values and a deterministic fallback chain; names and identifiers are not machine-translated. Numbers, dates and money are formatted by active locale, while stored values remain typed and locale-independent. Multi-currency records never sum amounts across currencies without an explicit conversion/reporting policy. Store event instants in UTC plus the relevant IANA timezone; attendance/academic dates are school-local dates, not UTC date truncations.

## Tenant placement move contract

Version one uses a **bounded maintenance-window move**, not dual writing or an unverified zero-downtime promise. It supports shared→dedicated, dedicated→shared, shared→shared and dedicated→dedicated. Same-driver moves ship before SQLite↔PostgreSQL typed conversion, which is a separately verified capability. No direct SQL dumps between engines.

Prerequisites: explicit operator authorization, source/target connectivity and identity, compatible schema versions, all module transfer manifests, capacity and region policy, verified backup, target absence of this tenant's records, storage/key access, sufficient maintenance window, and a stable placement version. Any missing prerequisite prevents the move before fencing.

The move record has stable move ID, tenant ID, source/target pool IDs, source epoch, next epoch, current state, version, initiating operator, timestamps, manifest/checksum reference, failure stage and recovery decision. Transition commands use compare-and-swap and are idempotent. Secrets and personal data are excluded from move logs.

| State | Observable behavior | Durable success condition / safe failure |
|---|---|---|
| requested / preflight | Source remains authoritative | Validated prerequisites and backup; failures leave source untouched |
| fencing | Tenant enters maintenance; new tenant reads/writes/jobs/uploads are refused or explicitly paused | Source transaction gate acquires exclusive lock after existing scoped transactions drain; source gate becomes frozen |
| copying | No new business writes; only operator status available | Versioned module manifests copy this tenant's rows, files, outbox/inbox and retained audit; resumable cursor; target gate remains closed |
| validating | Both copies frozen | Row counts, canonical per-table hashes, relationships, files/checksums, event deduplication, identities and schema receipts match |
| switching | Brief unavailability permitted | One control transaction CASes placement to target and increments epoch; source remains fenced |
| activating | Requests resolve target but may see maintenance | Target gate becomes writable only at the committed new epoch; stale source epochs stay rejected |
| observing | Tenant serves on target | Post-cutover smoke/readiness, worker ownership and audit checks pass; source retained read-prohibited for defined recovery period |
| completed | Target authoritative | Verification manifest and operator audit finalized; separate retention cleanup job may later delete source copy |
| failed / aborted | State-specific recovery only | Before pointer switch: target stays quarantined and source may be deliberately unfenced after verification. After switch: resume target activation or run a NEW reverse move; never simply repoint to a stale source |

Every transaction that reads/writes tenant business data passes a shared `TenantUnitOfWork` boundary that checks tenant, placement epoch and local gate in the same database transaction as the domain change. PostgreSQL gate locking coordinates active operations with the exclusive fence; SQLite transaction locking serializes the local gate with writes. The exact adapter lock protocol must be proven by concurrency tests before a move command is enabled. An application-side boolean or a cached routing lookup is insufficient.

Worker claims, delayed retries, scheduled jobs, export tasks, file publication and outbox consumers participate in fencing. A file upload stages bytes privately, then checks the current tenant gate before publishing metadata; an orphan cleanup handles failed publication. Delivery acknowledgements arriving during a move are durable and replayable with stable event/provider keys. No event ID is regenerated by a move. Network calls must not hold an unbounded database lock; delivery intent and provider idempotency manage the external boundary.

Reads after cutover must not return stale source data. Old processes and old pooled connections fail their epoch/gate checks, even if configuration or route cache is stale. Do not fall back to a shared database when a dedicated target is unavailable. Initial implementation resolves durable placement each operation; caching is permitted only with a tested invalidation/epoch protocol.

Transfer manifests are module-owned and typed: table, tenant discriminator, primary key, column types/version, dependency ordering, redaction policy for diagnostics and file references. CI rejects a new tenant table that has no manifest or an explicit non-transfer rationale. Cross-tenant foreign keys are invalid. Export/import uses stable ordered streaming and bounded batches. Resume never overwrites unrelated tenant data.

## Installation

Provision recognized foundation pools or empty targets. Verify receipts and pool ownership before serving. Whole-tenant relocation requires its own acceptance drill.

## Mandatory failure and concurrency tests

- Same tenant, different organization/school/campus/team: every read/write/list/search/export/document/report honors scope; explicitly granted group aggregates work.
- Cross-tenant IDs in nested resources, arrays, imports and references rejected even on a dedicated DB.
- Context switch while a request, modal, upload, export or worker callback is in flight; no old data rendered or written under the new context.
- Add/revoke grant, deactivate organization/school, move a campus, remove guardian relationship, alter team membership: stale sessions cannot retain expanded access.
- Race fence against a long transaction; fence waits/drains or safely times out. New commands cannot enter after freeze.
- Crash/restart at every move transition, including after control CAS before target activation; resume yields exactly one authoritative placement.
- Old API/worker process with old route, duplicate worker delivery, delayed provider acknowledgement and concurrent move requests: no split writer or duplicate effect.
- Target unavailable, disk full, checksum mismatch, schema mismatch, unsupported column type, missing file/key or wrong region: no cutover.
- Shared pool with at least two tenants: moving one preserves the other's data and availability; destination preexisting unrelated data unchanged.
- Same-driver and cross-driver moves in all supported directions; stable IDs, decimal/money/date/Unicode values, outbox receipts and blob hashes preserved.
- Post-cutover writes followed by rollback request: stale source never reactivated; a verified reverse move is required.
- Restore older control database against newer data pools: placement epochs/markers detect mismatch and fail closed.

## Implementation and release gates

Architecture/scope schema → compatibility migration → verified context/grants → all module transaction boundaries → real multi-context UI → transfer registry → fenced same-engine move → cross-engine converter → failure/restore drills. These gates precede dependent domain and release acceptance. No mutable `.env` tenant route is advertised as a migration tool. Restored/cloned pools require a controlled activation identity as well as their preserved marker; a copied marker alone is not proof of the unique current writer. Epochs never decrease.
