# Implementing a Goravel module

[Auth is the reference module](../modules/auth/README.md). Start there for the folder convention, role relationships, route registration, seeding and client resources.

Each Goravel module owns its models, services, database code, controllers and routes. The host registers the module. Web, Wails and Expo send versioned HTTP requests to the same application. The catalog is a list of available actions, and authorization is checked again at the handler.

## Follow a write

1. A client submits a bearer token to a versioned route. The identity middleware resolves the current membership and attaches its school and grants to the request.
2. The module controller decodes the DTO and checks the action's grants before passing a command to the application service.
3. The service enforces domain rules and calls a repository port. The repository resolves the verified school through the database catalog. Each SQL predicate includes its school even on a dedicated database.
4. A successful mutation writes its event to the outbox inside the same transaction. The outbox worker records durable delivery in the school's audit inbox. Versioned updates reject stale writes.
5. The client refreshes the affected query. A lost create response asks the user to check the register before repeating the write. Enrollment has an explicit idempotency key in its school, student and academic-year uniqueness rule.

## Where to change behavior

| Concern | Implementation point | Contract |
| --- | --- | --- |
| New workspace / translated title | Add one tab-separated row to `modules/catalog.tsv` | Group, stable module key, English title, French title |
| Module registration | `modules/<module>/module.go` and `modules/modulekit/definition.go` | Descriptor, repository binding and module-owned migration |
| Standard work list | `platform/workspace/service.go` and `modules/<module>/app/services/workspace.go` | School, entry ID, title, open/closed status and version |
| Replace standard storage | `modules/<module>/database/repository.go` | Implement `Get`, `List`, `Create` and versioned `Update`; preserve school scope and atomic event delivery |
| Specialized aggregate | `modules/students/app/models` or `modules/enrollment/app/models` | Value validation and state transitions live here |
| Cross-module query | `modules/students/app/contracts` | Enrollment consumes `Directory.Lookup` and shares the caller's transaction through the public contract |
| Additional schema changes | Module `database/migrations/`, registered in `bootstrap/migrations.go` | Give each new plan a new immutable ID and a down path. Run the deployment migrator across configured pools |
| Add a protected action | The owning module’s `routes/api.go` and `app/http/controllers/`, wired by the host | Version the request DTO, check action-specific grants on the server and register the path in `docs/api/openapi.yaml` |
| Shared UI | `clients/web/src/shell`, `src/components`, `src/features/collections`, `src/styles` | Layout, controls, loading and error states, and editable tables are shared |
| Specialized UI | `clients/web/src/features/specialized/ModulePanel.tsx` | Register a module panel; use the shared API client and scoped query cache |
| Operator-selected storage | `modules/documents/app/services/filesystem` | Local and S3 implementations share the same scoped blob-store port |

## Add or rename a workspace

Edit the manifest and regenerate with Python 3 and Go's `gofmt` on PATH:

```sh
python3 scripts/scaffold-modules.py
```

The generator creates missing module-owned files, regenerates display metadata and the central registry, and formats Go sources. Existing adapter implementations remain owner-maintained. Run the migrator before opening the new workspace. Its module page appears from the server catalog automatically. Stable keys also name tables and routes; changing a key for an existing module requires a migration and a client compatibility plan.

## Turn a work list into a business feature

A payroll work entry can track an open follow-up. A payroll posting needs its own amount, currency, ledger invariants, permission, idempotency and reversal policy. Introduce those types in Payroll's domain, define the application command and persistence port, wire the adapter and publish an explicit event. Keep the existing work-list page for coordination. Add a specialized panel when that command is available. Students, Enrollment and Documents provide three different examples of this extension.

## Runtime conventions

- The default control database holds memberships, sessions and tenant placement pins. School-specific repositories choose the physical connection from trusted operator configuration.
- API readiness checks the full migration ledger and each configured database's receipts. Use `go run ./cmd/migrate` or `make migrate` for deployment. A single deployment migrator should own schema changes at a time.
- The outbox worker handles delivery separately from HTTP. Its starter consumer writes the local audit inbox. External delivery adapters need their own idempotent consumer keyed by event ID.
- The Goravel scheduler submits the expired-session cleanup job daily at 02:00 UTC. Its default queue connection is synchronous. Setting `QUEUE_CONNECTION=database` enables Goravel's database-backed queue runner in the API process. Redis is available as infrastructure for future adapters.
- School permissions distinguish administrators, registrars and viewers. New teacher/class policies, guardian-child grants and domain-specific approval roles require explicit backend rules before exposing those workflows.
