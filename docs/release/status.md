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

The [instrumented iPhone run](https://github.com/elcomware/edupilot-os/actions/runs/42290820538/job/128143471806)
remained on the guest form after tapping and recorded no session POST at the
HTTPS fixture. On iOS, dragging the form now dismisses the keyboard. The drill
requires the keyboard to disappear before it taps Sign in.

The [iPhone form drill](https://github.com/elcomware/edupilot-os/actions/runs/42296795329/job/128164021506)
observed incomplete-form feedback although the native fields retained the
expected values after blur. The sign-in form now records the native field's
final edit event as well as each change and uses the submitted password text
for keyboard submission. The test requires edits to clear the initial
validation message before attempting the authenticated journey.

The [iPhone and Android simulator run](https://github.com/elcomware/edupilot-os/actions/runs/42303275286)
passed on the verification branch. On iPhone, tapping Sign in on the empty
form showed validation. After native credential entry, the keyboard's Go key
created a school session through HTTPS. A separate cold launch restored the
school, wrote the expected row and signed out. SQLite contained exactly one
matching workspace row and zero sessions afterward. Android passed using the
form button. Successful submission by tapping the populated iPhone form and
physical-device acceptance remain open.

The [native stack run](https://github.com/elcomware/edupilot-os/actions/runs/42308143293/job/128202618684)
stopped before browser execution because the pinned Playwright release could
not install Chromium on Ubuntu 26.04. Both jobs that install this browser
now select Ubuntu 24.04, including the optional manual Docker drill.

### Initialize a new shared tenant

On an installed native system, choose a fresh tenant UUID and confirm the shared
physical pool ID from the control database's `edu_pool_marker` row. Run:

```sh
go run ./cmd/provision-shared-tenant -tenant "$tenant" -pool-id "$shared_pool_id"
go run ./cmd/inspect-hierarchy -tenant "$tenant"
```

The command creates the placement pin, an active empty hierarchy at epoch 1
and version 1, and one creation event in the same database transaction. Use the
returned version when adding the first root organization. After an interrupted
response, inspect the same UUID before retrying. Existing placements require
inspection or recovery. Provisioning also rejects a configured dedicated route,
a different physical ID, an existing school realm, or foundation data for the
UUID in any configured pool.

Disposable PostgreSQL and SQLite checks exercised competing creates and an
outbox failure after the pin and gate inserts. The native SQLite CLI drill
provisioned a shared tenant, read it back, added its first organization, and
rejected a dedicated UUID. It verified the two expected events in shared storage
and an empty dedicated hierarchy. This command provisions storage only.
Dedicated initialization across databases, tenant profile and slug, user grants,
administrative screens and tenant moves remain open.

### Check a dedicated provisioning candidate

Run `go run ./cmd/check-dedicated-tenant -tenant "$tenant" -pool-id "$dedicated_pool_id"`
on an installed system before preparing a new dedicated placement. A successful
result identifies the tenant and the observed physical pool. The command checks
the configured route, migration receipts, pool markers, existing pins and school
realms. It checks all configured pools for this tenant's foundation records and
requires the dedicated destination to be empty of every owner's foundation data.
It performs no writes. Reservation, revalidation under a write fence, resumable
initialization and activation are separate release gates.

The [dedicated preflight verification](https://github.com/elcomware/edupilot-os/actions/runs/42318572352)
passed on disposable PostgreSQL and SQLite installations with three pools.
It rejected another school's row in the destination, the proposed tenant's
row in either other pool, a wrong marker and an existing pin. A separate
installed native SQLite CLI drill returned the expected JSON for the fresh
destination, left the pin table empty, and returned failure with no report
after a different school's synthetic row was added.

### Reserve a dedicated tenant

After native migration, run
`go run ./cmd/reserve-dedicated-tenant -tenant "$tenant" -pool-id "$dedicated_pool_id" -operation "$operation_uuid"`.
Generate a fresh random UUID for the operation and retain it with the returned
report. An exact retry returns the original creation time after rechecking the
configured destination and all foundation data. Conflicting operations fail.

The additive control migration stores a reservation journal. Reservation claims
the tenant's unique placement with an empty pool ID, claims its empty Auth realm,
revalidates the pools, and commits all three records together. Physical pool
markers require nonempty IDs, so first-use routing cannot claim or resolve the
pending tenant. Administrator seeding also loses its unique-realm claim and
refuses to bootstrap a user. The current resolver additionally rejects any
journaled reservation whose pin has been changed to a physical ID. Readiness
accepts valid reservations while checking their configured dedicated route.
Older binaries fail readiness while a reservation is pending.

This checkpoint deliberately leaves the tenant pending. Retain the operation
UUID and retry the same command after a transient failure. Do not clear its
journal, pin or Auth realm manually. Resumable target initialization, activation,
initial administrator enrollment and controlled cancellation are subsequent
dependencies. Whole-tenant moves and the full web, desktop and mobile
administration scope remain open.

Local verification passed `go test -count=1 -p 2 ./...`, `go vet -p 2 ./...`
and `go build -p 2 ./...`. The disposable SQLite fixture covers competing
reservations, reservation races against first-use routing and administrator
seeding, rejection of existing ownership/data, transaction rollback, exact retry,
routing denial, unrelated tenant access and installer replay. The same fixture
passed with Go race detection enabled.

A separate installed SQLite CLI drill built the preceding release's migrator
at `5f85f05`, installed fresh synthetic pools, upgraded with the additive
migration and reserved a new tenant. It verified exact JSON retry, conflicting
operation rejection, migration replay, older-binary fail-closed behavior and
zero users, tenant gates or provisioning events.


The [reservation checkpoint verification](https://github.com/elcomware/edupilot-os/actions/runs/42325605074)
passed on disposable PostgreSQL and SQLite. The workflow also passed client
unit tests/builds, desktop gateway tests, native Linux window smoke, Windows
compilation, browser journeys on the native stack and automated accessibility
checks. Independent code review found no remaining important issue after the
corrupt-pin resolver regression and the first-use/bootstrap race cases were added.
Device qualification and the pending initialization/activation workflow remain open.

### Prepare a reserved dedicated database

Run `go run ./cmd/prepare-dedicated-tenant -tenant "$tenant" -pool-id "$dedicated_pool_id" -operation "$operation_uuid"`
with the identifiers retained from reservation. The command holds the control
placement lock while committing a provisioning gate and one
`tenancy.tenant.prepared.v1` event together in the dedicated database. Its report
has `active=false`. The reservation and empty Auth realm continue to deny
routing and administrator bootstrap.

After an uncertain response, repeat this preparation command with the same
operation. It verifies the original gate, epoch, version and exact event payload.
An interruption before target commit leaves both records absent. An interruption
after target commit can be recovered by reading those durable records.
Use preparation for retries once the target has been initialized.

Local Go unit/feature checks, vet, build and race detection passed. The disposable
SQLite fixture covers concurrent preparation, exact retries, a wrong operation,
an injected outbox failure that rolls back the gate, corrupted gate/event
rejection, continued routing denial and healthy catalog readiness. A compiled
native CLI drill verified reservation followed by preparation and exact retry,
one provisioning gate/event and an empty control placement pin.
The [hosted preparation verification](https://github.com/elcomware/edupilot-os/actions/runs/42329921777)
passed the same fixture on PostgreSQL and SQLite, plus client unit tests/builds,
desktop gateway tests, native Linux window smoke, Windows compilation and
browser end-to-end and automated accessibility checks on the native stack.

Independent review confirmed the transaction boundary and corrected the report's
initial active flag. Activation must still reconcile target ownership and data,
publish the active placement safely and support interrupted recovery. Preparation
retry checks its own gate and event. Full target reconciliation, activation,
administrator enrollment, cancellation, tenant moves and all remaining UI and
release backlog criteria remain open.

### Reconcile the prepared target

Preparation now reconciles all owners' records in the installed foundation
tables inside its target transaction. After validating the exact gate and event,
it permits only those two records and the worker's optional matching audit
receipt. Unexpected workspace, hierarchy, event or audit records cause failure.
A legitimate event delivery remains compatible with preparation retry.

Local unit/feature checks, vet, build and Go race detection passed. The synthetic
SQLite fixture rejected workspace rows belonging to either the proposed tenant
or another owner, an unexpected organization, an extra event, an extra audit
receipt and a receipt whose tenant was corrupted. It also dispatched the real
prepared event through the worker and verified successful retry afterward.
A compiled native CLI drill rejected another owner's target row while preserving
that evidence and the pending reservation. The [hosted reconciliation checks](https://github.com/elcomware/edupilot-os/actions/runs/42334231569)
passed on PostgreSQL and SQLite, together with client tests/builds, native Linux
window smoke, Windows compilation and browser end-to-end/accessibility checks.
Independent review found no important issue in the transaction boundary.

This reconciliation covers the current installed foundation table inventory.
The release still requires transfer-manifest coverage for future module tables,
control-side enrollment reconciliation and recoverable activation with checks
held through cutover. Tenant moves, administration UI and the wider release
backlog remain open.

### Reconcile reserved enrollment before preparation

Preparation now asks Auth to verify the reserved realm in the existing control
transaction, after acquiring the placement lock and before writing the target.
The realm must exist and contain zero users, roles, permissions, role grants,
memberships or sessions. The Auth-owned contract keeps its enrollment table
inventory inside Auth. Any conflicting records are retained for investigation.

The regression fixture first reproduced preparation accepting an unexpected
role. SQLite checks now reject unexpected roles, permissions and users both
before initialization and on retry. They verify preservation of those records,
an untouched target after failed initial preparation, and acceptance when
only an unrelated realm contains a role. Foreign keys bind memberships, grants
and sessions to these enrollment roots.

Local unit/feature tests, vet, build and race detection passed. A compiled native
CLI drill rejected a populated reserved realm while preserving the role and
empty placement pin. Independent review found no important issue. The
[hosted enrollment reconciliation checks](https://github.com/elcomware/edupilot-os/actions/runs/42338650413)
passed on PostgreSQL and SQLite, together with client tests/builds, native Linux
window smoke, Windows compilation and browser end-to-end/accessibility checks.

This is control-side snapshot reconciliation for the installed Auth foundation.
Activation still requires explicit writer exclusion, revalidation through
cutover and recoverable placement publication. Initial administrator enrollment,
tenant moves, administration UI and the full release backlog remain open.

### Activate prepared dedicated storage

After applying native migrations, retain the tenant, physical pool and operation
UUID from reservation and preparation. Run the local operator command.

```sh
go run ./cmd/activate-dedicated-tenant -tenant "$tenant" \
  -pool-id "$dedicated_pool_id" -operation "$operation_uuid"
go run ./cmd/inspect-hierarchy -tenant "$tenant"
```

Activation requires the exact prepared gate at epoch 1 and hierarchy version 1,
its original event payload, the matching pending reservation and an empty Auth
realm. It reconciles the installed foundation inventory across configured pools.
The destination may contain only the prepared gate/event and their matching
optional worker audit receipt. Conflicting data stays available for investigation.

| Starting state | Successful transition or recovery |
| --- | --- |
| Reservation with an empty destination | Run preparation first. Activation requires its durable gate and event. |
| Prepared destination and pending reservation | Commit the active gate and one stable-ID activation event in the destination, then atomically store the control receipt, publish the physical placement pin and remove the reservation. |
| Active destination and pending reservation after interrupted publication | Repeat activation with the same three identifiers. Verify both original events, exact gate and all foundation/Auth data, then finish control publication. Routing stays denied until publication succeeds. |
| Published placement and completed receipt | Repeat activation with the same identifiers to return the immutable completion receipt. Normal hierarchy edits may have advanced the current hierarchy version. |
| Conflicting operation, pool, gate, receipt, event or enrollment data | Fail closed. Preserve the records and investigate the conflict before retrying. |

The placement row and Auth realm stay locked through target commit and control
publication. PostgreSQL uses READ COMMITTED with explicit row locks so enrollment
committed before the realm lock is visible to reconciliation. Its target table
locks fence the installed foundation and pool marker through target commit.
SQLite acquires its writer lock before reading these records. A target outbox
failure rolls back the gate transition. A later control failure leaves a pending
reservation that can be recovered with the same activation command. Continue
using activation for recovery once its target commit may have occurred.

The JSON receipt contains the original operation, event ID, physical pool,
placement epoch, initial hierarchy version and completion time. Use hierarchy
inspection for current hierarchy state. Completion retry still requires the
reserved Auth realm to be empty. Initial administrator enrollment requires a
subsequent dedicated protocol, and the existing bootstrap command continues to
reject that realm. Cancellation, tenant moves, administrative UI and the full
product and release backlog remain open.

Local verification covers concurrent activation, enrollment writers on the realm
fence, unexpected target/Auth records, event rollback, interrupted publication,
worker delivery, corrupted events/receipts, exact retry after a hierarchy edit,
pending routing denial and dedicated routing after publication. The complete Go
test suite, vet, build, feature race checks and a compiled disposable SQLite CLI
drill passed locally. An upgrade drill built the preceding release at
`51a6d9f`, retained its prepared reservation through the additive migration,
activated it, and verified exact retry after a hierarchy edit and migration replay.

The [hosted activation verification](https://github.com/elcomware/edupilot-os/actions/runs/42343978018)
passed on PostgreSQL and SQLite, plus client tests/builds, Linux native window
smoke, Windows compilation and browser end-to-end/accessibility checks.
The initial PostgreSQL run exposed a fixture cancellation race. The blocked
writer now uses an explicit transaction and server-confirmed lock timeout,
and completes rollback before the realm fence is released. Independent review
approved the protocol and fixture correction.

### Enroll the initial dedicated administrator

After native migrations and completed dedicated storage activation, retain the
same tenant UUID, expected physical pool ID and original operation UUID. Load
`EDUPILOT_SEED_EMAIL` and `EDUPILOT_SEED_PASSWORD` into the operator's environment
through the installation's secret handling, then run the local command.

```sh
go run ./cmd/enroll-dedicated-admin -tenant "$tenant" \
  -pool-id "$dedicated_pool_id" -operation "$operation_uuid"
```

The command rechecks the published pin, completed activation receipt, configured
route, physical owner, active epoch-1 gate and both original target events.
Auth verifies its reserved realm, seeds the existing roles and permissions, and
creates one administrator. The user, grants, immutable enrollment receipt and
`auth.admin.enrolled.v1` event commit together in the control database. The JSON
contains the tenant, pool, original operation, user ID, event ID and completion
time. Credentials come from the environment and remain absent from command
output, receipts and events. Database and flag failures use redacted diagnostics.

Repeat **enroll-dedicated-admin** with the same identifiers and credentials after
an uncertain response. An exact retry verifies the original user, current
password and a private, versioned commitment to the initial Auth state, then
returns the same receipt. Email, password, membership, role or grant changes,
additional members, deleted principals and orphan enrollment history cause
refusal and require investigation. Normal login and logout remain compatible
with this retry. Auth stores the commitment separately from the public receipt
and keeps the current bcrypt hash as the account's sole password verifier.

The placement and Auth fences stay held through verification and the control
commit. The dedicated target stays locked through that commit as well.
PostgreSQL takes table locks around Auth and the installed foundation
inventory so existing principal and grant updates are excluded alongside new
root inserts. These locks can delay other tenants' writers during credential
checks and target-lock waits. SQLite uses its database writer locks. The control pool may hold this tenant's enrollment
event and matching worker audit receipt. Other foundation data for this tenant
in control or another configured pool causes refusal.

Activation completion retry still requires an empty Auth realm. Once enrollment
has committed, use the enrollment command for its recovery and hierarchy
inspection for current storage state. This increment covers privileged native
initial enrollment. Scoped administrator interfaces, account lifecycle work,
tenant-school legacy mapping migration, cancellation, tenant moves and every
remaining production and client backlog requirement remain open.

Local verification passed the full `go test -count=1 -p 2 ./...` suite,
`go vet -p 2 ./...`, `go build -p 2 ./...` and feature race checks. The shared
both-engine fixture exercised preactivation/publication denial, corrupt
receipt/history/gate/placement, orphan Auth and audit history, failure after
Auth creation, competing credentials, exact retry, changed credentials/grants,
principal deletion, other-tenant isolation and real HTTP login, protected
access, cross-tenant denial and logout. SQLite passed locally. PostgreSQL was
skipped locally because the disposable CREATE DATABASE fixture was unavailable
and still requires hosted verification.

A compiled native SQLite CLI drill upgraded an activated installation from
`68f504f` with exactly one additive control migration. It verified enrollment,
identical retry, rollback after Auth creation, secret-redacted SQL and flag
failures, matching audit receipt retry, refusal after unrelated control data,
activation retry refusal after enrollment and migration replay. The original
target schema and historical activation events were preserved.
