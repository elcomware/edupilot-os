# EduPilot OS starter

The repository starts from Goravel v1.18.0 and keeps its facades, configuration, bootstrap, Artisan and migration conventions. EduPilot OS uses strict Go module boundaries, a versioned HTTP API, a shared React web/desktop shell, and a React Native mobile client. SQLite and PostgreSQL share domain contracts. School-owned queries always include the authenticated school, including on dedicated databases.

The following checkpoints describe the original starter phase. Current implementation and verification follow the [production release plan](superpowers/plans/2026-11-06-production-release.md) and [release board](release/status.md).

## Checkpoints

| Iteration | Deliverable |
| --- | --- |
| 01 | Repository baseline and acceptance plan |
| 02 | Standard Goravel application and pinned Go dependencies |
| 03 | Platform contracts, tenant placement and transaction boundaries |
| 04 | Core platform module packages |
| 05 | Student lifecycle module packages |
| 06 | Academic module packages |
| 07 | Finance module packages |
| 08 | Operations, engagement and intelligence module packages |
| 09 | Validated module composition and public contracts |
| 10 | SQL persistence, migrations and reliable outbox |
| 11 | Identity, sessions, permissions and administrator provisioning |
| 12 | Students reference domain and application |
| 13 | Authorized API, Students storage and runtime composition |
| 14 | Shared TypeScript API and client workspace |
| 15 | Customizable web shell and shared page components |
| 16 | Connected web pages and client unit specifications |
| 17 | Wails and Expo entry points |
| 18 | Deployment, setup and worker processes |
| 19 | Feature, architecture, integration and end-to-end specifications |
| 20 | Cross-package review, operational handoff and adapter inventory |

A successful handoff includes a concrete implementation for shared infrastructure, a working-path design from sign-in to school-scoped Student mutations, a module-owned adapter boundary for every other module and honest API descriptions of available actions. During this phase the owner requested test authoring before execution. Production-phase checks now run on pushes and pull requests.
