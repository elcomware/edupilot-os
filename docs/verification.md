# EduPilot OS verification

The starter phase deliberately deferred tests. The production phase now runs available checks and records unavailable checks separately. This is **not a production-readiness declaration**: the full product backlog, expanded tenancy implementation and release gates remain open.

## Executed baseline — 6 November 2026

| Scope | Command / exercise | Observed result |
|---|---|---|
| Native supervisor | `python3 -m unittest discover -s scripts/tests -v` | 13 tests passed; independent review approved after process-ownership regression fix |
| Go domain, feature, architecture and SQLite integration | `go test -p 4 -json ./...` | Exit 0: 33 passing test/subtest events; four service-dependent test skips |
| Web shared client/components/preferences | `npm run test:web` | 14 tests passed |
| Mobile contracts and recovery | `npm run typecheck --workspace @edupilot/mobile`; `npm run test --workspace @edupilot/mobile` | TypeScript passed; 9 tests passed. Runner uses `node --import tsx --test` to avoid CLI IPC requirements |
| Desktop gateway | In `clients/desktop`: `go test -json ./internal/gateway` | 14 tests passed |
| Native stack | Disposable SQLite/file roots and keys; `python3 scripts/dev.py` | Migrations, binary builds, API, outbox worker and loopback Vite started successfully; supervisor stopped on termination |
| Real API end to end | Native `scripts/ci-e2e.sh` | Three cases passed: school isolation, enrollment retry/withdrawal and private document round trip |
| Browser end to end | Same suite, system Chromium | Two cases blocked before assertions: this executor denies Chromium's Unix socket creation (`EPERM`). They remain required in CI/on a supported host |
| Expo Android/iOS bundles | `npm run export --workspace @edupilot/mobile` | Both JavaScript bundles exported; this does not verify signed binaries or physical devices |

[Machine-readable baseline evidence](release/evidence/2026-11-06-baseline.json) identifies the baseline and limitations. Native-launcher review is complete. Dependency hardening is tracked separately in the release status.

## Explicit unavailable checks

| Check | Why unavailable here | How to run it |
|---|---|---|
| `TestHybridRoutingAndTransactionalDelivery/postgres` | No PostgreSQL executable or disposable DSN | Provision a native disposable PostgreSQL database and set `EDUPILOT_TEST_POSTGRES_DSN` |
| `TestRepositoryTransactions/postgres` | Same missing service | Same DSN, `make test-go` |
| `TestEnrollmentAcrossModuleBoundaries/postgres` | Same missing service | Same DSN, `make test-go` |
| `TestS3PrivateBucketRoundTrip` | No dedicated test bucket/credentials | Configure the isolated bucket variables described by the test; never use a real document bucket |
| Chromium UI journeys | Sandbox denies required Unix sockets | Run Native verification CI or `make e2e` on a browser-capable host |
| Optional Docker stack | Docker executable unavailable | Explicit `make docker-up [DB=postgres]`; optional manual workflow |
| Native windows/devices/signing | No supported GUI/device/signing setup | Follow Wails/Expo release matrix and record actual device/installer evidence |

## Continuous verification

The **Native verification** workflow runs on pushes to main/release branches, pull requests and manual dispatch. It installs PostgreSQL natively for its integration job, runs Go checks, client tests/builds, native launcher tests and the native browser stack. The existing **Manual verification** workflow retains an explicitly optional Docker matrix. Workflow definitions are not evidence of a successful remote run; link actual run results before closing a release gate.

Use `make test-native test-go test-web test-mobile test-desktop` for local suites. Browser setup and disposable accounts are documented in [tests/e2e/README.md](../tests/e2e/README.md). Test imports, migrations, moves and recovery only on disposable data. Every new multi-tenant/group workflow adds sibling-school and cross-tenant cases; the starter's school isolation test alone does not verify the new hierarchy.

The [security and hosted-run follow-up](release/evidence/2026-11-06-dependencies.md) records the first native PostgreSQL and browser results, the isolated fixture fix, and the updated dependency scans.
