# Local deployment

## Start the stack

Use Python 3, Make, and Docker Compose v2 (with BuildKit). From the repository root:

```sh
make setup
make up                        # SQLite (one API, one worker)
make seed-api                  # prompts for a real school ID, email, password
make ps
```

Open [the web app](http://127.0.0.1:8080), then sign in with the account you provisioned. API health is available at `http://127.0.0.1:3000/api/v1/health`; readiness at `/api/v1/ready` checks service dependencies. The web server forwards `/api/` to the API on the private Compose network. API, web, object service, console and PostgreSQL host ports bind only to loopback. Redis has no published host port.

Setup fills blank or missing keys with random local values, writes a private `.env`, and preserves existing nonempty settings. It never creates a user or a sample record. For unattended admin provisioning pass `EDUPILOT_SEED_SCHOOL`, `EDUPILOT_SEED_EMAIL` and `EDUPILOT_SEED_PASSWORD` in the calling process environment (not as command-line literals). Exact re-provisioning is safe; a different password does not take over an existing account.

## Database modes

| Mode | Start | Provision | Stop (retain data) |
|---|---|---|---|
| SQLite | `make up` | `make seed-api` | `make down` |
| PostgreSQL | `make up DB=postgres` | `make seed-api DB=postgres` | `make down DB=postgres` |

The base file stores SQLite in a named volume at `database/edupilot.sqlite`; API, migrations and worker share it. The PostgreSQL overlay starts PostgreSQL 16, changes the app connection, and waits for database readiness before migration. Stop the current mode before switching. The two modes have independent persisted stores; switching does not copy data between them. Run `make migrate` or `make migrate DB=postgres` to apply pending migrations explicitly. Both commands run the Goravel migrator through `./cmd/migrate`, which exits with failure if a migration or readiness check fails. Changing PostgreSQL credentials in `.env` does not rotate credentials in an initialized database.

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
| `worker` | Independent process polling the durable SQL outbox. `make logs` includes its output. |
| `web` | Serves the built React assets via nginx. Configure `WEB_PORT` (default 8080). |
| `postgres` | PostgreSQL mode only; configurable `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PORT`. |
| `redis` | Persistent Redis service at `redis:6379` for future cache/queue adapters. |
| `storage` | MinIO S3-compatible service; API on 9000, console on 9001. |

Student records and the outbox use SQL. Documents store bytes in the shared runtime volume by default (`EDUPILOT_DOCUMENTS_DRIVER=local`, `EDUPILOT_DOCUMENTS_ROOT=storage/documents`). Redis is available for optional adapters; the outbox does not depend on it.

To store documents in MinIO, use the generated `MINIO_ROOT_USER` and `MINIO_ROOT_PASSWORD` to open the [object console](http://127.0.0.1:9001). Provision a private bucket and a separate key with only GetObject/PutObject for `schools/*/documents/*`. In `.env` set `EDUPILOT_DOCUMENTS_DRIVER=s3`, `S3_BUCKET`, `S3_ACCESS_KEY_ID`, and `S3_SECRET_ACCESS_KEY`, then rerun `make up` with the same database mode. For Compose MinIO the S3 endpoint is `http://storage:9000` and region defaults to `us-east-1`; use `S3_ENDPOINT` and `S3_REGION` to override. The app receives no MinIO root credentials. Existing files do not move when switching drivers. See [document storage](document-storage.md) for its contract.

The Compose topology targets a single development machine. For a hosted deployment, terminate TLS, use a secret manager, use PostgreSQL with verified TLS, back up both shared and dedicated stores, and supervise the API and worker independently. Do not distribute this Compose file with a live `.env`.

## Native development and explicit verification

The API uses Go 1.25+, CGO/toolchain support for SQLite, and port 3000 by default. Use `make native-migrate`, `make native-seed`, then start `make native-api` and `make native-worker` in separate terminals. For the React web client use Node 24, `make install` and `make dev-web`; Vite proxies `/api/` to port 3000. Mobile and desktop have their own platform prerequisites.

Automated checks are opt-in: `make test-go`, `make test-web`, `make test-mobile`, and `make e2e`. See [live browser setup](../tests/e2e/README.md) for the two-school workflow. For PostgreSQL adapter tests, set `EDUPILOT_TEST_POSTGRES_DSN` to an isolated test database on which the test user may create temporary schemas. The manual GitHub Actions workflow exercises both database modes. Nothing runs on push or pull request.
