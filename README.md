# EduPilot OS

A school operating system starter built on Goravel 1.18. The repository contains 42 registered modules, a versioned HTTP API, one React workspace for the web and Wails desktop, and an Expo mobile client. School data lives in SQLite or PostgreSQL. New records come from the API and persist in the selected school's database.

## Start a school workspace

From a fresh clone, install Docker with Compose v2, Make and Python 3. Then run:

```sh
make up
make seed-api
```

Open [the school workspace](http://127.0.0.1:8080) and sign in with the school ID, email and password entered at the prompt. This starts the SQLite configuration and runs migrations before serving requests. For PostgreSQL, use `make up DB=postgres` followed by `make seed-api DB=postgres`.

## What you can do

| Area | Starter behavior |
| --- | --- |
| Identity | Provision an administrator, add members, assign school roles, sign in, revoke a session and check current permission grants. |
| Students | Register a student, page through the active register, edit names, and archive a record with version checks. |
| Enrollment | Place an active student in an academic year and grade, retry the same placement safely, or withdraw it. Enrollment uses the public Students directory in the school transaction. |
| Documents | Upload a supported file, retrieve its ID and download its bytes from private local storage or a configured S3-compatible bucket. |
| Module workspaces | All 41 modules other than Students have their own persistent work list, schema and repository binding. These entries track follow-up work. Financial postings, report-card rules, timetabling algorithms and other subject-specific workflows belong in their respective modules. |
| Layout | Choose language, theme, density, accent and reading width, then pin modules for the current user and school. Web and desktop share the shell, table, form and state components. |
| Operations | Run migrations with receipt checks, route selected schools to dedicated databases, and deliver transactional outbox events to the audit inbox. |

The browser keeps its bearer in memory. Reloading requires signing in again. The mobile client stores its bearer in device secure storage and uses the same API. Desktop connects to an operator-configured API origin.

## Develop and extend

| Task | Start here |
| --- | --- |
| Configure services, accounts and database modes | [Local deployment](docs/local-deployment.md) · [Tenant placement](platform/database/README.md) |
| Add a module, use case, or adapter | [Adapter guide](docs/adapter-guide.md) · [Students](modules/students/README.md) · [Enrollment](modules/enrollment/placements/README.md) |
| Inspect the API | [OpenAPI 3.1 contract](docs/api/openapi.yaml) |
| Change all workspaces or add one specialized panel | [UI design](DESIGN.md) · [Workspace behavior](UX-CONTRACT.md) · [Web client](clients/web/README.md) |
| Build native clients | [Wails desktop](clients/desktop/README.md) · [Expo mobile](clients/mobile/README.md) |
| Configure files or async delivery | [Document storage](docs/document-storage.md) · [Outbox worker](platform/outbox/README.md) |
| Find files and deferred checks | [Source tree](docs/source-tree.txt) · [Verification handoff](docs/verification.md) |

Set `EDUPILOT_ENABLED_MODULES` to a comma-separated list to restrict the catalog and its routes for a deployment. The default includes all modules. Enable `students` whenever you enable `enrollment`.

The implementation extends the standard Goravel v1.18.0 tree, including its Artisan entry points, app facades, providers, migrations and dependency pins. The upstream MIT license is preserved in [LICENSE](LICENSE).
