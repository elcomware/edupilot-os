# Local deployment

## Start natively

Install Go 1.25+, Python 3, Make, and Node matching the root `package.json` engines: `^22.13.0 || ^24.3.0 || >=26` (Node 24.3+ is recommended). Native supervision requires a POSIX Python build with `waitid`/`WNOWAIT` support (commonly Linux or WSL; availability varies on macOS). `make check-native` verifies that capability before any service is started. The normal startup path needs no Docker or PostgreSQL server. From the repository root:

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

Use `make docker-ps` or `make docker-logs`, with the same `DB=postgres` selector if applicable, to inspect that stack. Open the container [web app](http://127.0.0.1:8080). Its nginx server forwards `/api/` to the API on the private Compose network. API, web, object service, console and PostgreSQL host ports bind only to loopback. Redis has no published host port. Stop the native stack before starting the Compose stack on the same API port.

The base Compose file stores SQLite in a named volume at `database/edupilot.sqlite`; API, migrations and worker share it. The PostgreSQL overlay starts PostgreSQL 16, changes the app connection, and waits for database readiness before migration. Stop the current mode before switching. Native SQLite, container SQLite and container PostgreSQL have independent persisted stores; switching does not copy data between them. Both migration paths run `./cmd/migrate`, which exits with failure if a migration or readiness check fails. Changing PostgreSQL credentials in `.env` does not rotate credentials in an initialized database.

Shared schools require no placement entry. To add a dedicated school, configure `EDUPILOT_DEDICATED_DATABASES` as one JSON array in `.env`, for example:

```dotenv
EDUPILOT_DEDICATED_DATABASES='[{"tenant":"north","driver":"sqlite","database":"database/north.sqlite"}]'
```

For a dedicated PostgreSQL database, provision the database and its login before configuring a mapping with `tenant`, `driver`, `database`, `host`, `port`, `username`, `password`, and `sslmode`. Inside Compose use a service DNS name (such as `postgres`), not the host's loopback address. Migrate before serving; account/session tables remain in the control database. Do not change a live school's mapping without a separate data migration and backup.

## Services and extensions

| Service | Purpose |
|---|---|
| `migrate` | Runs Go command `./cmd/migrate`, checks migration history and pool receipts, and exits successfully before API and worker start. |
| `api` | Goravel HTTP service. Configure the published port with `API_PORT` (default 3000). |
| `worker` | Independent process polling the durable SQL outbox. `make docker-logs` includes its output. |
| `web` | Serves the built React assets via nginx. Configure `WEB_PORT` (default 8080). |
| `postgres` | PostgreSQL mode only; configurable `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PORT`. |
| `redis` | Persistent Redis service at `redis:6379` for future cache/queue adapters. |
| `storage` | MinIO S3-compatible service; API on 9000, console on 9001. |

Student records and the outbox use SQL. Documents store bytes in the shared runtime volume by default (`EDUPILOT_DOCUMENTS_DRIVER=local`, `EDUPILOT_DOCUMENTS_ROOT=storage/documents`). Redis is available for optional adapters; the outbox does not depend on it.

To store documents in MinIO, use the generated `MINIO_ROOT_USER` and `MINIO_ROOT_PASSWORD` to open the [object console](http://127.0.0.1:9001). Provision a private bucket and a separate key with only GetObject/PutObject for `schools/*/documents/*`. In `.env` set `EDUPILOT_DOCUMENTS_DRIVER=s3`, `S3_BUCKET`, `S3_ACCESS_KEY_ID`, and `S3_SECRET_ACCESS_KEY`, then rerun `make docker-up` with the same database mode. For Compose MinIO the S3 endpoint is `http://storage:9000` and region defaults to `us-east-1`; use `S3_ENDPOINT` and `S3_REGION` to override. The app receives no MinIO root credentials. Existing files do not move when switching drivers. See [document storage](document-storage.md) for its contract.

The Compose topology targets a single development machine. For a hosted deployment, terminate TLS, use a secret manager, use PostgreSQL with verified TLS, back up both shared and dedicated stores, and supervise the API and worker independently. Do not distribute this Compose file with a live `.env`.

## Clients and explicit verification

`make dev-web` starts the local-only React development server independently. Mobile and desktop have their own platform prerequisites. The API uses port 3000 by default.

Automated checks are opt-in: `make test-native`, `make test-go`, `make test-web`, `make test-mobile`, and `make e2e`. `scripts/ci-e2e.sh` provisions temporary schools against a running native stack by default; set `EDUPILOT_E2E_MODE=docker` for the optional container stack. It defaults to web port 5173 for native and 8080 for Docker, with `E2E_BASE_URL` available to override either. See [live browser setup](../tests/e2e/README.md) for the two-school workflow. For PostgreSQL adapter tests, set `EDUPILOT_TEST_POSTGRES_DSN` to an isolated test database on which the test user may create temporary schemas. The manual GitHub Actions workflow exercises both container database modes. Nothing runs on push or pull request.
