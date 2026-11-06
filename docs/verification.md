# EduPilot OS verification

Run local checks from the repository root. `make test-native test-go test-web test-mobile test-desktop` exercises the supervisor, Go unit/feature/integration suites, React components, Expo logic and desktop gateway. `make up` starts the native stack. The optional container matrix has a separate manual workflow. [Live browser setup](../tests/e2e/README.md)

## Reproducible evidence

| Scope | Latest evidence and boundaries |
|---|---|
| Shared and dedicated SQL | Disposable SQLite and PostgreSQL fixtures exercise the original starter, additive group schemas, correct physical ownership, replay, read-only placement inspection and canonical frozen-source content inventory. [Foundation evidence](release/evidence/2026-11-06-tenancy-foundation.md) |
| Browser journeys | Six native-stack journeys exercise API school isolation, enrollment retry/withdrawal, private document access, student register actions, preferences and pinned navigation. The desktop and 320px French layout captures contain synthetic fixtures. [Hosted run and captures](https://github.com/elcomware/edupilot-os/actions/runs/42148038227) |
| Dependencies | The security checkpoint scanned reachable Go source and the compiled API, and audited the root and E2E Node dependency trees. CI checks the current checkout again. [Scan results](release/evidence/2026-11-06-dependencies.md) |
| Historical starter baseline | The initial native run and its original limitations are preserved for comparison. [Baseline results](release/evidence/2026-11-06-baseline.json) |

The combined local Go run recorded 159 passing test/subtest events and 11 service-dependent skips. Ten need PostgreSQL fixtures, including one database-creation opt-in. One needs a private S3 test bucket. Local vet and build passed. Both the SQL inspector and document verifier were exercised with the race detector. A disposable native end-to-end run also passed all three API journeys.

The web component suite passed 34 tests, and the mobile suite passed ten. Web production build, both Expo JavaScript exports, desktop gateway tests and the Windows GUI cross-build passed. See the [release board](release/status.md) for hosted Linux window, Chromium and accessibility evidence.

## Service and device requirements

| Check | Where it runs |
|---|---|
| Native PostgreSQL integration | CI provisions an isolated PostgreSQL instance. For local schema-level tests set `EDUPILOT_TEST_POSTGRES_DSN` on a disposable database with schema creation privileges. The compiled CLI test additionally needs `EDUPILOT_TEST_POSTGRES_CREATE_DATABASES=1` and a role permitted to create isolated databases. See [operator commands and fixture requirements](operations/tenant-foundation-upgrade.md). |
| Chromium against real APIs | The hosted runner launches Chromium. The local executor rejects the browser's required Unix socket before assertions. Run `make e2e` on a supported host with disposable accounts. |
| Private S3 provider access | `TestS3PrivateBucketRoundTrip` needs an isolated private bucket and scoped test credentials. [Provider setup](document-storage.md) |
| Optional Compose | `make docker-up` and `make docker-up DB=postgres` are optional developer paths. This local executor has no Docker executable. |
| Release clients and deployment | Record device interaction, screen-reader testing, signed installers, store builds, backup/restore and operator acceptance against the named deployment target. |

The [release board](release/status.md) records the current feature and qualification boundaries. The [20-sprint plan](superpowers/plans/2026-11-06-production-release.md) specifies the additional sibling-school and cross-tenant journeys required as group workflows enter the runtime.
