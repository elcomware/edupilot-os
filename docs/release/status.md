# Production release status

## Installed foundation

The installed tree contains Core, Auth and Tenancy. The school directory offers Auth and Tenancy. Auth owns users, school roles, permissions and sessions. Core supplies shared contracts, infrastructure and UI resources under `modules/core/resources`. Native setup is the default.

Add business modules as their workflows are implemented. Keep users, roles and permissions inside Auth. Register services through Goravel IoC, define migrations with Goravel Blueprint, and register web and native pages at build time. Verify module isolation, access controls, both databases and each client journey.

## Open work

1. Finish group hierarchy administration and scoped sessions for multiple organizations, schools, campuses, teams, curricula, languages and locations.
2. Verify backup, restore and whole-tenant moves with fencing, resumable copy, reconciliation and rollback on this schema.
3. Deliver the remaining domain workflows and their client experiences against the [release backlog](../superpowers/plans/2026-11-06-production-release.md) and [UI matrix](../product/frontend-matrix.md).
4. Qualify installation, physical devices, accessibility, security and operator acceptance.

The installer accepts empty databases and recognized foundation pools. Drills use disposable data. Production deployment and signed distribution require separate authorization.

Installer checks on disposable SQLite and PostgreSQL pools cover clean installation,
replay and rejection of an unrelated table or a view named `migrations` before the
control ledger is created. SQLite also rejects a user table named `sqlitex_records`.
The live HTTP check verifies school isolation and confirms that sign-out revokes access.

### iPhone workspace verification

The foundation's [iPhone simulator run](https://github.com/elcomware/edupilot-os/actions/runs/42246292166/job/127989261329) reached the Tenancy editor but timed out waiting to see the saved title and status. The revised XCTest checks the entered title, waits for keyboard dismissal, activates the accessible submit button, and checks the save acknowledgement before scrolling to the persisted row. The fixture still requires exactly one expected database row and zero sessions after sign-out. The local disposable SQLite/PostgreSQL suite passed all 39 Go tests and subtests, and all 10 mobile unit tests passed. Simulator acceptance requires a successful hosted run.

The [instrumented iPhone test](https://github.com/elcomware/edupilot-os/actions/runs/42254178788/job/128016831767) isolated a rapid-input failure before submission: the native field contained `Ciew` after typing `Campus review`. The mobile workspace now seeds the native editor from its saved draft once per mount and resets it after a confirmed create. Native typing, navigation drafts, and server acknowledgement are still exercised by their respective checks.
