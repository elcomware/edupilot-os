# EduPilot OS release execution status

Plan: [Production release backlog](../superpowers/plans/2026-11-06-production-release.md). Spec: [Production scope](../product/production-scope.md).

## Baseline

- Starter: `ae7a791cdbbdc1bf76be7f1a166e9a226c3da177`.
- Original 20 starter checkpoints pushed. Build checks passed; original instruction deferred tests.
- New authorization: write/run tests, plan and implement the full product including full professional UI, native-first operation and optional Docker, then push implementation checkpoints.
- Product naming ruling: **EduPilot OS** in copy, **edupilot-os** in technical identifiers. The provider is now `EduPilotOS`; tenant and school are distinct entities; existing conflation is a migration task.
- Release state: **not production ready**. Full domain backlog and external qualification gates remain open.

## Board

| Increment | State | Evidence / next action |
|---|---|---|
| Plan / Task 1 | Verified/pushed | Remote commits `2c72280`, `2e03da6`; all 42 modules mapped to full UI stories; independent scope reviews incorporated |
| S01 Native/test baseline | In progress | Native launcher and dependency checks reviewed. Hosted SQLite and PostgreSQL baseline passed; [patched native CI](https://github.com/elcomware/edupilot-os/actions/runs/42143303446) passed all three jobs, including full Chromium journeys and PostgreSQL |
| S02 Group tenancy + full frontend foundation | In progress | Scope/legacy migration, authorized context UI, inherited customization, multilingual records and browser qualification |
| S03–S17 Domain delivery | Planned | Story-by-story backend + full UI + tests; see dependencies in plan |
| S18 Operations + tenant mobility | Planned | Fenced shared/isolated/cross-engine moves, native packaging, backup/restore, upgrade and observability |
| S19 Clients | Planned | Mobile feature completion and actual signed native release checks |
| S20 Qualification | Planned / external gates | UAT, deployment/provider credentials, signing and hardware evidence required |

## Decisions

- Native SQLite is the simplest default; PostgreSQL remains supported. Containers are explicitly optional.
- Implement enabling staff, workflow and document slices before dependent domain stories; no placeholder satisfies a dependency.
- Existing user authorization covers implementation and pushes, so proceed after publishing the plan without an extra approval checkpoint.
- Use isolated branch `release/production-readiness` locally and lease-protected publication to the already-authorized repository branch.

- Group tenancy ruling: tenant is the customer/isolation/placement unit; schools, campuses, teams and curricula are scoped children/associations. Entire tenants move together. School ownership transfers are separate.
- Initial tenant moves use bounded maintenance with database-enforced fencing and epochs; no dual-write/zero-downtime promise.
- Architecture and frontend plan reviews requested independently; critical scope gaps incorporated before implementation.

## Evidence

See [verification](../verification.md) and [baseline evidence](evidence/2026-11-06-baseline.json). Chromium cannot launch here (Unix socket EPERM), PostgreSQL and S3 services are unavailable. Both backend jobs passed on [hosted native CI](https://github.com/elcomware/edupilot-os/actions/runs/42140789673); client job passed four of five journeys. The preference reload exposed a shared test identity hitting the real login limit.

[Dependency and fixture evidence](evidence/2026-11-06-dependencies.md): patched source and compiled API scans report zero affected vulnerabilities; root and E2E npm audits report zero. No release candidate is tagged.

The pure topology model resolves tenant, organization, school and campus ancestry, inherited active status and a persisted hierarchy revision. Snapshot tests exercise sibling schools, conflicting campus context, missing owners, team selection, optional ID serialization and independent snapshots. Production sign-in and placement still use the starter school scope. Multi-school activation waits for the offline upgrade, grant checks, scoped transaction boundary and end-to-end qualification in the [tenant implementation plan](../superpowers/plans/2026-11-06-tenant-foundation.md).

The starter fixtures exercise 42 registered modules on shared, dedicated and mixed placements, including clean and deliberately damaged records. [Ownership inventory](../architecture/tenant-ownership-inventory.md).

The new schema appends 48 plans to the original migrations. On a shared control/data pool plus one dedicated pool, that creates 94 additional migration registrations. Control retains identity and placements. Data pools enforce composite tenant/school ownership for topology, student and enrollment records, documents, outbox and all 41 work lists. Upgrading the schema creates empty tables. A separate verified offline upgrade is required before switching sessions or business reads to them. [Foundation table inventory](../architecture/foundation-ownership-inventory.md).

Personal navigation ordering and reset are implemented in the shared web/desktop shell. Users can move pinned modules with a keyboard, reload their order, and reset this workspace's device settings. Account changes isolate language, pins, theme and spacing. The reset confirmation keeps focus and announces the result. Automated web tests and the production build pass. Saved view/column presets and group inheritance remain open under UI-03 and CFG.

[Tenancy and UI verification record](evidence/2026-11-06-tenancy-foundation.md). Offline copy/resume is the next tenancy implementation dependency.

Tasks 1–3 of the tenant plan have passed hosted SQLite, PostgreSQL and browser qualification. Task 4 has begun with [offline mapping validation](../operations/tenant-foundation-upgrade.md). Source inspection, manifest preparation, copy, resume and post-copy verification remain open.
