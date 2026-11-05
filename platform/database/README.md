# Shared and dedicated database placement

The control connection holds identity and routing pins. It is also the default tenant database. A dedicated school gets its own physical database, but its identity and membership remain on the control connection. The API must authorize membership before calling the catalog, and repositories must still include `tenant_id` in every data predicate on every pool.

Use `DB_CONNECTION=sqlite` and a writable SQLite `DB_DATABASE` path for local development. Use `DB_CONNECTION=postgres` with `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`, and `DB_SSLMODE` for PostgreSQL. The PostgreSQL default SSL mode is `require`; production deployments should configure certificate verification. Database files, parent directories and PostgreSQL database/user permissions must be provisioned before migrating. Use the same settings for API, worker, migration runner and admin seeder.

`EDUPILOT_DEDICATED_DATABASES` is a JSON array. It defaults to `[]`. Example with both drivers:

```json
[
  {"tenant":"west","driver":"sqlite","database":"database/west.sqlite"},
  {"tenant":"north","driver":"postgres","host":"db.internal","port":5432,"database":"school_north","username":"edu_north","password":"<from secret store>","sslmode":"verify-full"}
]
```

Only the configuration selects physical targets. Each database gets one ownership marker and an immutable pool ID. First use pins a school to that ID in the control database. A missing override for an existing dedicated school or an unexpected physical replacement fails closed. Do not repoint or reuse production databases as a shortcut for moving students.

Run `./artisan migrate` once per deployment before API or worker rollout. Each registered plan has a signature per configured connection; Goravel tracks those signatures in its normal migrations table on the control connection. Adding a new dedicated database under a new tenant runs all tenant plans on that database at the next migration. Each plan stores a receipt alongside its DDL on the target pool. If a migration process stops after DDL commit but before Goravel updates its default-connection history, a retry reconciles history without rerunning that DDL. Keep the receipts table with database backups and never truncate it independently. Identity-only plans never run on dedicated databases.

Moving an existing school requires a scheduled cutover: quiesce its writes and workers, back up both databases, apply the schema to the intended target and transfer the school's data, pending outbox and audit history. Verify counts and IDs, then update the control routing pin to the verified target pool ID and update all process configurations before allowing traffic. Keep the old database intact for rollback. No automatic copy or online migration is provided. Replacing a database under the same connection name also requires a reviewed restoration or migration-ledger reconciliation; changing the DSN alone does not replay already recorded migrations. A fresh or restored control database must be restored together with its routing pins.
