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

The [native mobile verification](https://github.com/elcomware/edupilot-os/actions/runs/42264975684) passed on iPhone and Android at `aab9110`. The iPhone journey waited for the keyboard and acknowledged each injected character once. It verified cold-launch school identity, the complete displayed title and open status, and sign-out across relaunch. Both fixtures verified one matching SQLite row and zero remaining sessions. Android passed on retry after an initial launch timed out before the login field appeared. Rapid iPhone typing still needs a separate reproduction and physical-device qualification remains open.

### Hierarchy integrity

Hierarchy restoration rejects a snapshot whose version is too low for its
persisted organizations, schools and campuses. The native installation drill
now exercises the installed foreign keys and unique constraints with two
synthetic tenants. Both SQLite and PostgreSQL accepted the same codes under
different owners and rejected duplicate sibling codes and cross-tenant parent
references. All 40 Go tests and subtests and the native HTTP school-isolation
journey passed locally. Hierarchy administration and scoped sessions remain open.

Follow-up iPhone runs failed at different directory labels immediately after the keyboard submitted sign-in. The login inputs now retain native text and submit their latest values without sending a controlled value back on every keystroke. The simulator fixture waits for the real scoped session in disposable SQLite; the cold-launch UI check verifies the restored school. The exact workspace title, status, durable row and revocation checks remain required. Mobile type checking and all ten mobile unit tests passed locally.

### Physical pool identity

Readiness and migration preflight require distinct physical IDs across all
configured, initialized pools. The disposable SQLite and PostgreSQL drills
copied the shared ID onto a dedicated database with the correct owner label.
Both checks rejected it, and the missing control ledger entry stayed absent.
Restoring the dedicated ID allowed the interrupted migration to reconcile.
This covers configured pools at inspection time. Writer fencing, verified
copy and cutover are still required for tenant moves.

The [Linux desktop drill](https://github.com/elcomware/edupilot-os/actions/runs/42264165326/job/128051578796)
intermittently reached a populated table without finding the expected edit
button after typing. The driver now waits for native keyboard focus and checks
the synthetic title in WebKit before submitting. The window, accessible edit
button, screenshot and independent database assertions still have to pass.

### Inspect a persisted hierarchy

Operators can run `go run ./cmd/inspect-hierarchy -tenant TENANT_UUID -campus CAMPUS_UUID`
against a configured installation. The command checks the placement catalog and
reads the selected tenant gate, campus, school and organization ancestors from a
consistent snapshot. Its JSON includes the hierarchy version and whether the
selected path is active. The optional `-organization` and `-school` flags can
select a parent directly or require a campus to belong to that parent.

The disposable SQLite and PostgreSQL drills verified root-first ancestry,
an inactive parent, conflicting ancestors, a cross-tenant lookup, a cycle,
a wrong gate pool ID and a lookup that left an unpinned tenant unregistered.
The compiled CLI also returned the expected JSON on a synthetic SQLite pool.
This is a local operator lookup for existing UUID-based hierarchies.
Provisioning, versioned edits and user grant checks remain open.

### Add an organization to an existing hierarchy

A local operator can add a root organization or a child of an active organization
in an initialized, pinned tenant. Supply a new UUID, the expected placement epoch
and the current hierarchy version. For example:

```sh
go run ./cmd/add-organization -tenant "$tenant" -organization "$new_org" \
  -parent "$parent_org" -name "Regional office" -epoch "$epoch" -version "$version"
```

Omit `-parent` for a root. The command returns the committed path and new version.
After an interrupted response, inspect the same organization UUID before retrying.
The organization, version increment and outbox event commit in one transaction.

Disposable PostgreSQL and SQLite drills each accepted exactly one of two
concurrent edits at the same version. They rejected an old epoch, a replay,
a maintenance gate and an inactive parent. An injected outbox failure rolled
back the organization and version. A separate native CLI drill read the result
back from dedicated SQLite and verified that only the dedicated pool contained
the organization and event. The native HTTP school-isolation journey passed.
Initial tenant provisioning, user-facing administration, scoped grants and
verified tenant moves remain open.

The [organization checkpoint's iPhone run](https://github.com/elcomware/edupilot-os/actions/runs/42277337262/job/128108074496)
twice finished keyboard submission without creating the expected school session.
The iPhone login drill now dismisses the keyboard and taps the visible Sign in
button. The independent SQLite session check,
cold-launch school check, workspace write and revocation checks still apply.

The [follow-up iPhone job](https://github.com/elcomware/edupilot-os/actions/runs/42283470468/job/128118171172)
stopped at keyboard dismissal before it could tap Sign in. iPhone login and
workspace interaction now use XCTest. The login test checks the school and
email as typed and checks only the length of the secure password field before
tapping Sign in. The runner checks the persisted session before starting the
separate cold-launch workspace test. Only named fixture screenshots are uploaded.

On the [first XCTest login run](https://github.com/elcomware/edupilot-os/actions/runs/42287046326/job/128130451801),
all field checks and the button tap completed, but the school label did not
appear. The synthetic fixture now serves HTTPS through a loopback Go reverse
proxy and records the HTTP status of session creation. The iPhone uses the
certificate's IPv4 loopback address and waits for the authenticated Sign out
control before the independent database and cold-launch identity checks.
