# EduPilot OS starter

The repository starts from Goravel v1.18.0 and keeps its facades, configuration, bootstrap, Artisan and migration conventions. The school OS uses strict Go module boundaries, a versioned HTTP API, a shared React web/desktop shell, and a React Native mobile client. SQLite and PostgreSQL share domain contracts. School-owned queries always include the authenticated school, including on dedicated databases.

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

A successful handoff includes a concrete implementation for shared infrastructure, a working-path design from sign-in to school-scoped Student mutations, a module-owned adapter boundary for every other module and honest API descriptions of available actions. No push-triggered test jobs. All automated tests are authored for later execution at the owner's request.
