# EduPilot OS

A school operating system starter built on Goravel 1.18. The repository contains 42 registered modules, a versioned HTTP API, one React workspace for the web and Wails desktop, and an Expo mobile client. School data lives in SQLite or PostgreSQL. New records come from the API and persist in the selected school's database.

## Start a school workspace

Install Go 1.26.10 or newer, Python 3, Make, and Node matching `package.json` engines (Node 22.13+, 24.3+, or 26+; Node 24 is a good default). Native supervision requires a POSIX Python build with `waitid`/`WNOWAIT` support (commonly Linux or WSL; availability varies on macOS). From a fresh clone run:

```sh
make install
make up
```

`make up` checks prerequisites, creates a private local configuration, migrates SQLite, builds the API and outbox worker, and supervises both alongside Vite in the foreground. Open [the school workspace](http://127.0.0.1:5173). In a second terminal, provision an administrator:

```sh
make seed-api
```

Sign in with the school ID, email and password entered at the prompt. Press Ctrl-C in the `make up` terminal to stop its services. `make check-native` checks prerequisites without changing data or starting services. Use `make migrate` to apply migrations separately.

Docker Compose is optional: `make docker-up` and `make docker-seed` start and provision the container stack at [127.0.0.1:8080](http://127.0.0.1:8080). Add `DB=postgres` to these commands for the PostgreSQL overlay, and use `make docker-down DB=postgres` to stop that mode. See [local deployment](docs/local-deployment.md) for configuration and operational details.

## What you can do

| Area | Starter behavior |
| --- | --- |
| Identity | Provision an administrator, add members, assign school roles, sign in, revoke a session and check current permission grants. |
| Students | Register a student, page through the active register, edit names, and archive a record with version checks. |
| Enrollment | Place an active student in an academic year and grade, retry the same placement safely, or withdraw it. Enrollment uses the public Students directory in the school transaction. |
| Documents | Upload a supported file, retrieve its ID and download its bytes from private local storage or a configured S3-compatible bucket. |
| Module workspaces | All 41 modules other than Students have their own persistent work list, schema and repository binding. These entries track follow-up work. Financial postings, report-card rules, timetabling algorithms and other subject-specific workflows belong in their respective modules. |
| Layout | Choose language, theme, density, accent and reading width, reorder pinned modules, and choose visible columns in the student register, module work lists and enrollment placements. Preferences are scoped to the current user and school on the device. Web and desktop share the shell, table, form and state components. |
| Operations | Run migrations with receipt checks, route selected schools to dedicated databases, and deliver transactional outbox events to the audit inbox. |

The browser keeps its bearer in memory. Reloading requires signing in again. The mobile client stores its bearer in device secure storage and uses the same API. Desktop connects to an operator-configured API origin.

## Release plan and present limits

The [Scrum backlog](docs/superpowers/plans/2026-11-06-production-release.md) and [release status](docs/release/status.md) distinguish delivered workflows from planned features. Today the starter scopes records to a school. The [group tenancy specification](docs/product/tenancy-and-placement.md) defines the verified hierarchy, shared/dedicated placement and controlled tenant moves; those workflows are not yet enabled.

## Develop and extend

Clone `elcomware/edupilot-os` and use `github.com/elcomware/edupilot` as the import prefix for root Go packages.

| Task | Start here |
| --- | --- |
| Configure services, accounts and database modes | [Local deployment](docs/local-deployment.md) · [Tenant placement](platform/database/README.md) |
| Build a module or trace a user’s access | [Auth reference](modules/auth/README.md) · [Module guide](docs/adapter-guide.md) · [Enrollment](modules/enrollment/providers/README.md) |
| Inspect the API | [OpenAPI 3.1 contract](docs/api/openapi.yaml) |
| Change all workspaces or add one specialized panel | [UI design](DESIGN.md) · [Workspace behavior](UX-CONTRACT.md) · [Web client](clients/web/README.md) |
| Build native clients | [Wails desktop](clients/desktop/README.md) · [Expo mobile](clients/mobile/README.md) |
| Configure files or async delivery | [Document storage](docs/document-storage.md) · [Outbox worker](platform/outbox/README.md) |
| Find tracked files and verification results | [Source tree](docs/source-tree.txt) · [Verification handoff](docs/verification.md) |

Set `EDUPILOT_ENABLED_MODULES` to a comma-separated list to restrict the catalog and its routes for a deployment. The default includes all modules. Enable `students` whenever you enable `enrollment`.

The implementation extends the standard Goravel v1.18.0 tree, including its Artisan entry points, app facades, providers, migrations and its module layout. The [dependency evidence](docs/release/evidence/2026-11-06-dependencies.md) documents the pinned upstream compatibility fix. The upstream MIT license is preserved in [LICENSE](LICENSE).
