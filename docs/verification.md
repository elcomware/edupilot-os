# Verification handoff

The owner requested that tests be authored and left for a subsequent test pass. No unit, feature, database integration, browser or native test suite was executed during this implementation. CI runs only on manual dispatch.

## Build evidence

| Target | Work completed during implementation |
| --- | --- |
| Goravel API, migrator, seed and worker | Go production packages compiled, and go vet completed for the root module, including its test source. |
| React web | TypeScript checking and the Vite production bundle completed. |
| Expo mobile | TypeScript checking completed. Expo exported JavaScript bundles for Android and iOS. |
| Wails desktop | The shared web assets were embedded and the Windows amd64 desktop executable cross-compiled. Native execution and installer signing are deferred. |

## First test pass

Start in a disposable deployment. Each command below is an explicit test action for the next session.

| Scope | Command or target | Evidence to review |
| --- | --- | --- |
| Backend | `make test-go` | Domain invariants, school isolation, HTTP mappings, optimistic concurrency, transactional outbox, native migrations, package import boundaries, and every module work list |
| PostgreSQL | Set `EDUPILOT_TEST_POSTGRES_DSN` to a disposable database and run `make test-go` | PostgreSQL adapters, migration replay, placement isolation and rollback. Tests create disposable schemas |
| Web | `make test-web` | Shared client, preference scope, conflict recovery, editor and specialized feature behavior |
| Mobile | `make test-mobile` | Endpoint configuration, session transitions and TypeScript contracts |
| Wails gateway | From `clients/desktop`: `go test ./internal/gateway` | Origin restrictions, redirect rejection, byte limits and document response mapping |
| End to end | `make e2e-install`, provision two temporary schools, then `make e2e` | Sign-in, student lifecycle and reload, preference isolation, enrollment retries, binary file round trips and cross-school denial |
| Full stack on both drivers | Dispatch the **Manual verification** workflow | Compose startup on SQLite and PostgreSQL plus the preceding server/web/browser checks |

Reference the [Playwright configuration](../tests/e2e/README.md) for disposable account variables. Inspect native window launch, device behavior, keyboard navigation, print, and visual layout during a separate hands-on pass. This handoff records build results only; there is no test pass rate or coverage measurement yet.
