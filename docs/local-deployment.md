# Local deployment

## Start natively

Install Go 1.26.10+, Python 3, Make, and Node matching the root `package.json` engines: `^22.13.0 || ^24.3.0 || >=26` (Node 24.3+ is recommended). Native supervision requires a POSIX Python build with `waitid`/`WNOWAIT` support (commonly Linux or WSL; availability varies on macOS). `make check-native` verifies that capability before any service is started. The normal startup path needs no Docker or PostgreSQL server. From the repository root:

```sh
make install
make check-native              # optional, read-only dependency check
make up                        # foreground API, outbox worker and local-only Vite
```

Then, in another terminal:

```sh
make seed-api                  # prompts for school ID, email and a hidden password
```

Open [the web app](http://127.0.0.1:5173) and sign in with the account you provisioned. Vite binds to `127.0.0.1:5173` with strict port selection and proxies `/api/` to the API on port 3000. API health is available at `http://127.0.0.1:3000/api/v1/health`; readiness at `/api/v1/ready` checks service dependencies. If you change the API port, set `EDUPILOT_API_PROXY` to its origin for Vite.

`make up` checks tools and installed client dependencies, runs setup and migrations, then builds the API and worker in a private, git-ignored directory under `tmp/`. Both binaries and Vite belong to the foreground supervisor. A failed migration or build stops startup; an exited service stops its siblings. Ctrl-C gracefully stops the owned process groups, with a bounded forced cleanup for unresponsive processes. `make down` prints this native stop instruction; it never invokes Docker. Logs stream in the startup terminal. The launcher requires POSIX process groups and waitable child leaders with `waitid`/`WNOWAIT`. Completed preparation groups are cleaned and retired before services start; leaders stay unreaped until their final group signal so numeric group IDs cannot be reused for unrelated sessions during cleanup. Hosts lacking this capability receive guidance before any child starts; use a supported Python/OS, WSL, or the explicit Docker commands below.

Setup fills blank or missing keys with random local values, writes a private `.env`, and preserves existing nonempty settings. It never creates a user or a sample record. For unattended admin provisioning pass `EDUPILOT_SEED_SCHOOL`, `EDUPILOT_SEED_EMAIL` and `EDUPILOT_SEED_PASSWORD` in the calling process environment (not as command-line literals). Passwords are not printed. Exact re-provisioning is safe; a different password does not take over an existing account.

SQLite is the default, stored at `database/edupilot.sqlite`. `make migrate` applies pending native migrations explicitly. For native PostgreSQL, provision a database separately and configure `DB_CONNECTION`, `DB_DATABASE`, `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD` and `DB_SSLMODE` in `.env`. Native commands read `.env`; the Make `DB=postgres` selector applies only to Docker commands. `make native-migrate` and `make native-seed` remain aliases; `make native-api` and `make native-worker` run individual services for debugging.

## Optional Docker Compose stack

Install Docker with Compose v2 and BuildKit. These explicit targets preserve the container path:

| Mode | Start | Provision | Migrate | Stop (retain data) |
|---|---|---|---|---|
| SQLite | `make docker-up` | `make docker-seed` | `make docker-migrate` | `make docker-down` |
| PostgreSQL | `make docker-up DB=postgres` | `make docker-seed DB=postgres` | `make docker-migrate DB=postgres` | `make docker-down DB=postgres` |

Use `make docker-ps` or `make docker-logs`, with the same `DB=postgres` selector if applicable, to inspect that stack. Open the container [web app](http://127.0.0.1:8080). Its nginx server forwards `/api/` to the API on the private Compose network. Published API, web and PostgreSQL ports bind to loopback. Stop the native stack before starting the Compose stack on the same API port.

The base Compose file stores SQLite in a named volume at `database/edupilot.sqlite`; API, migrations and worker share it. The PostgreSQL overlay starts PostgreSQL 16, changes the app connection, and waits for database readiness before migration. Stop the current mode before switching. Native SQLite, container SQLite and container PostgreSQL have independent persisted stores; switching does not copy data between them. Both migration paths run `./cmd/migrate`, which exits with failure if a migration or readiness check fails. Changing PostgreSQL credentials in `.env` does not rotate credentials in an initialized database.

Shared schools require no placement entry. To add a dedicated school, configure `EDUPILOT_DEDICATED_DATABASES` as one JSON array in `.env`, for example:

```dotenv
EDUPILOT_DEDICATED_DATABASES='[{"tenant":"north","driver":"sqlite","database":"database/north.sqlite"}]'
```

For a dedicated PostgreSQL database, provision the database and its login before configuring a mapping with `tenant`, `driver`, `database`, `host`, `port`, `username`, `password`, and `sslmode`. Inside Compose use a service DNS name (such as `postgres`), not the host's loopback address. Migrate before serving; account/session tables remain in the control database. Do not change a live school's mapping without a separate data migration and backup.


See [verification](verification.md) for database and client checks.

## Add schools and campuses

Use a trusted local operator account on an initialized, pinned tenant. Retain
fresh UUIDs for the school and campus, the expected placement epoch and the
hierarchy version returned by the latest provisioning or hierarchy command.
Create the owning organization first with `cmd/add-organization`.

```sh
go run ./cmd/add-school -tenant "$tenant" -organization "$organization" \
  -school "$school" -code MAIN -name "Central School" \
  -epoch "$epoch" -version "$version"
```

Use the returned `hierarchyVersion` as `school_version` for the campus command.

```sh
go run ./cmd/add-campus -tenant "$tenant" -school "$school" \
  -campus "$campus" -code NORTH -name "North Campus" -timezone Europe/Paris \
  -epoch "$epoch" -version "$school_version"
go run ./cmd/inspect-hierarchy -tenant "$tenant" -campus "$campus"
```

School codes are unique within their owning organization. Campus codes are
unique within their owning school. Codes are trimmed and uppercased, contain
1–64 ASCII letters, digits, periods, underscores or hyphens, and start with a
letter or digit. Names are trimmed and accept up to 200 Unicode characters,
excluding control characters. A campus requires an IANA timezone such as
`Europe/Paris`, `Africa/Nairobi` or `UTC`.

Each command checks configured placement readiness, validates the complete
active ancestry and commits one row, one hierarchy-version increment and one
creation event in the tenant's pinned pool. The JSON response contains the
committed path and version. A stale epoch/version, closed tenant gate, invalid
parent, duplicate identity/code or failed event write exits unsuccessfully.
Input errors use exit status 2. Operational conflicts and failures use status 1.

After an uncertain response, inspect the retained school or campus UUID before
retrying. A successful edit consumes its expected version, so a replay conflicts.
Auth grants, sessions and user-facing hierarchy administration require their
separate workflows. Scoped grants, lifecycle/reparenting, additional hierarchy
kinds and verified tenant relocation are pending release dependencies.
