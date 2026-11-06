# EduPilot OS

Goravel school administration for web, Wails desktop and Expo mobile. The foundation installs Auth and Tenancy. Core supplies shared infrastructure.

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

## Where to work

| Responsibility | Start here |
|---|---|
| Users, school roles, grants, credentials, HTTP and gRPC | [Auth](modules/auth/README.md) |
| School topology, pool ownership and school work lists | [Tenancy](modules/tenancy/) |
| Reusable Go contracts, events, scope, presentation and database primitives | [Core](modules/core/) |
| Installed services and migrations | [Bootstrap](bootstrap/) |
| Welcome and admin layouts, controls, native UI and page registration | [Shared resources](resources/README.md) |

The API enforces school scope on every protected request. My account lists the current school role and effective permissions. The Schools workspace manages titled entries with version checks. The [release board](docs/release/status.md) tracks full hierarchy administration and other planned workflows.
