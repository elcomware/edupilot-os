# Prepare the group tenancy upgrade

The current release checkpoint can prepare a reconciled frozen SQL+object source, validate a proposed mapping file offline, inspect legacy SQL placement and run a final read-only mapping/source reconciliation. Use the native command from the repository root:

```sh
go run ./cmd/tenant-upgrade -mode=validate-manifest -manifest=/private/upgrade.json
```

A successful run prints the manifest version, number of schools and SHA-256 fingerprint. It opens only the supplied file. A malformed file exits with status 1. Missing arguments or an unsupported mode exit with status 2. Keep the mapping file in private operator storage.

## Mapping file

The example below illustrates the format. For a real upgrade, the source digest must come from the frozen source inventory. The physical marker must match the recorded pool marker. Allocate the four target UUIDs once and keep them with the approved manifest.

```json
{
  "version": 1,
  "sourceSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "schools": [
    {
      "legacySchool": "north",
      "sourcePool": "sqlite",
      "sourceMarker": "example-source-marker",
      "tenant": "b0421bcb-122a-4e2d-8294-261864c2e249",
      "organization": "c88e6562-3dd9-4d83-a115-eacd8c791069",
      "school": "558a7688-4335-4b37-b9ca-91b7a5842cc4",
      "campus": "42a8b3dd-1da0-4dcf-a0af-11140e68fa68",
      "timezone": "Europe/Paris",
      "locale": "fr"
    }
  ]
}
```

Each starter school receives its own tenant at upgrade. Creating additional schools in a group is a later, explicitly authorized provisioning operation. The validator checks that source names and markers agree across mappings, target UUIDs are unique, each school has an explicit IANA timezone and initial UI locale, and the JSON uses the expected fields exactly once. Reordering school entries preserves the fingerprint. Changing a mapping, timezone, locale or source digest changes it. One file can contain up to 4,096 schools and 4 MiB of JSON.

A successful validation covers the submitted file. Database inspection, a verified backup, maintenance fencing, copy with per-pool receipts and verification against the frozen source are separate requirements. Operators should keep the current runtime in school scope. The [release board](../release/status.md) records which upgrade phases have passed.

## Inventory a frozen SQL source before mapping

Stop all API, worker, seed and migrator processes and freeze every configured SQL pool before running this command. Keep the freeze in place until the inventory and independently retained backup evidence have been approved. The command obtains a repeatable, read-only snapshot on each pool; separate pools do not form one atomic cross-database snapshot.

Create a mode `0600` selection file containing the complete, sorted list of legitimate historical school scopes. It contains no target UUIDs or final mapping:

```json
{
  "version": 1,
  "schools": ["north", "south"]
}
```

If only a verified local-document inventory exists, its version 1 aggregate can be bound without exposing object keys. This remains preflight evidence and is not ready for copy because the exact legacy reference set has not been reconciled:

```json
{
  "version": 1,
  "schools": ["north", "south"],
  "documents": {
    "version": 1,
    "schools": 2,
    "documents": 37,
    "contentBytes": 184302,
    "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  }
}
```

Then run:

```sh
chmod 600 /private/source-selection.json
go run ./cmd/tenant-upgrade -mode=inventory-source -selection=/private/source-selection.json > /private/source-inventory.json
```

The selection is limited to 4 MiB and 4,096 unique, strictly sorted legacy school slugs. Exact JSON fields are required and duplicate fields are rejected. The command accepts no target identities, arbitrary SQL, DSNs or source paths. It uses the same validated native SQLite/PostgreSQL configuration as the server, with native installation as the default; Docker is not required.

Every original physical pool is bound by its configured name and original singleton marker content. The report inventories 55 control tables and 47 tables on each dedicated pool: native migration history, local migration receipts, routing metadata, identity, jobs, Students, enrollment, outbox/audit and all 41 registered workspace tables. Rows are ordered by their declared primary keys. Canonical values preserve NULL versus empty, Unicode and binary bytes, normalize integers and UTC timestamps, and produce per-schema, per-table, per-pool and overall SHA-256 digests. The report contains counts and digests only, not row values, primary keys, student IDs, credentials, object keys, connection strings or file/database paths. It does not repeat the selected school list, but configured pool names can encode historical school names and remain sensitive operational metadata, so keep the report private.

Exit 0 requires exact schema compatibility, complete reads, an exact match between selected schools and the placement inventory, no unknown school scope, and no placement issue. Exit 1 emits either a redacted configuration diagnostic or a JSON report with `complete=false`, issue codes and no overall digest. An absent SQLite database is not created. Never use an incomplete report as `sourceSha256`.

Version 1 reports are classified `frozen-source-sql` and never set `readyForCopy=true`, including when they bind a version 1 document aggregate. Preserve them as preflight evidence; do not place their digest in a copy manifest.

## Assemble authoritative SQL+object source evidence

First complete the reference reconciliation in [Inventory a frozen starter document backup](legacy-local-document-inventory.md). Its successful version 2 result contains counts plus three digests and no object keys. Copy those exact fields into a version 2 source selection:

```json
{
  "version": 2,
  "schools": ["north", "south"],
  "documents": {
    "version": 2,
    "schools": 2,
    "references": 37,
    "documents": 37,
    "contentBytes": 184302,
    "referencesSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "inventorySha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "sha256": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
  }
}
```

The example hashes are format placeholders only; a real selection must use the retained reconciler output. The parser recomputes the reconciled digest from the counts and component hashes and rejects mismatches, missing fields, duplicate fields, unequal reference/document counts or a version 2 selection without document evidence.

Run the same `inventory-source` command while the SQL pools and document backup still describe the approved freeze. A successful version 2 result is classified `frozen-source-sql-objects`, has `readyForCopy=true`, and binds each per-pool SQL digest, the exact reference-set digest, the independently approved version 1 byte-inventory digest and the reconciled document digest. Only that complete report's top-level `sha256` is the authoritative source fingerprint for the later mapping/copy/verification stages. The command does not itself prove the deployment freeze, copy data or activate tenants; backup/restore proof, maintenance fencing, copy/resume and post-copy verification remain required.

## Reconcile the mapping before prepare

Keep the SQL pools and document backup frozen. Set the mapping's `sourceSha256` to the successful version 2 source inventory digest, retain the exact private version 2 selection used to generate it, and run:

```sh
chmod 600 /private/upgrade.json /private/source-selection.json
go run ./cmd/tenant-upgrade \
  -mode=dry-run \
  -manifest=/private/upgrade.json \
  -selection=/private/source-selection.json > /private/upgrade-dry-run.json
```

`dry-run` regenerates the complete SQL+object inventory from the current read-only sources, re-inspects legacy placement, and compares every mapping pool and physical marker with native configuration. It requires the mapping and selection to contain exactly the same sorted legacy-school set. Version 1 or incomplete document evidence, a changed source digest, missing/extra school, routing issue, pool mismatch or marker mismatch fails closed.

The successful report is classified `legacy-upgrade-dry-run`, sets `complete=true` and `readyForPrepare=true`, and contains only school/pool counts plus the approved manifest and source digests. It omits legacy school names, target IDs, markers, object keys, row values, paths and credentials. Failure reports contain redacted issue codes and no approval digests. Input files must be private regular files; a missing SQLite source is not created.

This is still a read-only precondition, not the mutation phase. It does not acquire a deployment maintenance lock, verify a backup/restore drill, create journal or tenant rows, copy data, activate tenants or fence old binaries. Do not release the freeze or treat `readyForPrepare` as permission to run an unimplemented prepare/copy command. Durable prepare/resume remains blocked until the operator-confirmed maintenance mechanism can prove the old API, worker, seed and migrator processes are stopped and their source access is fenced.


## Rehearse the SQL write fence on an offline restore

Use a disposable restore with synthetic schools and private mapping and selection files. Stop the API, worker, seeder and migrator before the rehearsal. Point native configuration at the restore and use a role that can lock the registered PostgreSQL tables or open the restored SQLite files for writing. Run the following command under the same maintenance window as the document inventory.

```sh
umask 077
go run ./cmd/tenant-upgrade -mode=rehearse-sql-fence \
  -manifest=/private/upgrade.json -selection=/private/source-selection.json \
  > /private/sql-fence-rehearsal.json
```

The command acquires all configured pools in a stable order. SQLite reserves the database writer with `BEGIN IMMEDIATE`. PostgreSQL takes `EXCLUSIVE` table locks with `NOWAIT` on the original 55 control/shared tables and the 47 legacy tables on each dedicated pool. Ordinary readers may continue. An existing writer on any required target aborts the rehearsal and releases earlier locks. The service verifies the mapping, active outbox leases, pool markers, placement and approved source fingerprint through those exact transactions, then rolls back and closes every connection. The operation has a two-minute deadline and never creates a missing SQLite database.

Exit 0 emits `legacy-sql-fence-rehearsal` with the pool count and approved digests. Exit 1 covers lock contention, insufficient privilege, missing schema or source, inconsistent evidence, cancellation and output failures. A failed run emits no approval digests. Stop or drain the conflicting writer before retrying. Locks last only for this invocation. The report grants no copy or activation authority.

The SQL check reuses the supplied, previously reconciled document evidence. Deployment maintenance must separately prevent legacy processes from restarting, fence file/object writes and database replacement, and verify a restorable backup. PostgreSQL coverage is limited to registered legacy tables. Journal preparation and copying will require their own continuously held maintenance boundary.

## Verify an isolated restore with local documents

Stop all writers to the restored copy and keep its document tree immutable throughout verification. Point native database settings at the restored control and every restored dedicated pool. Use the original approved mapping, version 2 source selection and independently preserved reference list from **before** the backup. Rebuilding those inputs from the restore would defeat the comparison.

```sh
go run ./cmd/tenant-upgrade -mode=verify-local-restore \
  -manifest=/private/mapping.json -selection=/private/source-selection.json \
  -references=/private/object-references.json -root=/private/restored-documents
```

The command re-reads the actual envelopes and content bytes, reconciles the exact reference set and compares its aggregate with the approved evidence. It then computes the restored SQL fingerprint and verifies school membership, placement and pool markers through read-only connections. Exit 0 returns the pool and document counts and the approved hashes. An unavailable pool, missing object or fingerprint mismatch returns exit 1 without an approval report. This qualifies the contents of an isolated copy with local document storage. Use the native PostgreSQL drill below to exercise logical archive capture and restore on synthetic data. A real installation still requires a continuous maintenance boundary and a rehearsal of its own backup and storage provider.

## Rehearse PostgreSQL archive recovery

Use a disposable PostgreSQL test instance and its matching native client utilities. Configure the private test DSN as described under PostgreSQL CLI acceptance below. The test login needs `CREATEDB`. Run from the repository root:

```sh
pg_dump --version
pg_restore --version
EDUPILOT_TEST_POSTGRES_CREATE_DATABASES=1 \
  go test -count=1 -v ./database/edumigrations -run '^TestLegacyPostgresArchiveRestore$'
```

Look for a **PASS** on the named test. Without both the DSN and the database-creation opt-in, Go reports a skip. With both set, a missing archive utility fails the test. Allow space for four temporary databases, two private archives and the copied document envelopes. The test removes its own databases on cleanup.

The drill creates two synthetic schools using the frozen starter migrations. One school uses the shared control/data pool and one uses a dedicated database. Before backup it computes the SQL and local-document evidence and writes the private mapping, selection and exact object references. With fixture writers quiescent, it captures each database using `pg_dump --format=custom`, copies the documents to a separate tree, and replays each archive into a fresh database using `pg_restore --single-transaction --exit-on-error`.

The compiled verifier must refuse the restore while the dedicated database is empty. After both archives have been restored, it must reproduce both pre-backup hashes. Updating a single restored student row must make it exit 1 with empty standard output. The drill uses the fixture login for ownership and the target's default tablespace. Production role grants, tablespaces, managed-provider snapshots, point-in-time recovery, external document storage and enforcement of the maintenance window require deployment-specific qualification.

## Inspect legacy SQL placement

From the native deployment directory, with the same environment and `.env` used by the application:

```sh
go run ./cmd/tenant-upgrade -mode=inspect-placement > /private/placement-inspection.json
```

The mode must be explicit. A blank/default mode refuses to run. `inspect-placement` takes no manifest; `validate-manifest -manifest=PATH` remains an offline file validator; `inventory-source -selection=PRIVATE_JSON` is the separately frozen content inventory above; `dry-run` requires both private mapping and selection files. Native APP_KEY/configuration validation and process-environment-over-`.env` precedence are preserved. This command does not boot HTTP, run migrations, dispatch events, provision routes or create placement pins. It reads DB_CONNECTION/DB_DATABASE and the native PostgreSQL settings plus EDUPILOT_DEDICATED_DATABASES, using the application's validated connection names and legacy owner strings. PostgreSQL uses the same native public search_path; no arbitrary DSN/SQL input is accepted.

SQLite sources must already exist as regular files and are opened with `mode=ro` and `query_only=1`; an absent source is never created. SQLite uses deferred read transactions, and PostgreSQL uses read-only repeatable-read transactions plus a read-only connection default. PostgreSQL credentials are used only to connect; reports/diagnostics omit passwords, DSNs, hosts, database/file paths, physical marker IDs, user IDs, emails, password/token hashes, payloads and record bodies. School scope IDs and configured connection names remain sensitive operational metadata: keep the report private.

The JSON classification is always `inspect-only`. `pools` contains per-table row counts for Students, enrollment placements, outbox/inbox and all 41 registered work-list tables, including empty tables. `schools` identifies known legacy school scopes, their identity/pin presence, the pool expected by current configuration and observed per-pool/table counts. Markers, control identity_schools and original placement pins are read to reconcile routing. ExpectedPool describes configuration; it does not authorize a mapping or repair a route.

The report is deterministically sorted. Exit 0 means every required SQL source/table was read and no routing issue was found. Exit 1 means reported issues, incomplete inspection, invalid/unavailable native configuration, or output failure. Invalid arguments/unsupported modes exit 2. A configuration failure emits a generic diagnostic; database/source failures emit redacted issue codes. `complete=false` means counts may be partial/lower bounds and missing identity/pin values may be unknown. Never interpret partial output as proof that a source is empty. Limits are 128 configured pools, 4,096 school scopes, 8,192 catalog tables per pool and a 30-second overall deadline; crossing a limit fails incomplete.

| Issue code | Meaning |
| --- | --- |
| invalid_configuration / invalid_workspace_registry | Native settings or the reviewed 41-work-list registry are invalid. |
| pool_unavailable / schema_unavailable | A configured read-only connection or catalog could not be inspected. |
| unexpected_control_table | A dedicated pool contains a legacy control/identity/infrastructure table; only its catalog name is inspected. |
| missing_legacy_table | A required original table is absent; inspection is incomplete. |
| routing_query_failed / scope_query_failed | A required routing/count query failed; inspection is incomplete. |
| invalid_pool_marker / wrong_pool_owner | The original singleton marker is absent/malformed or its recorded owner conflicts with the configured connection. |
| duplicate_pool_marker / duplicate_pool_owner | Multiple configured targets claim the same physical marker or owner. |
| unpinned_school / duplicate_school_pin / invalid_school_pin / unknown_pin_pool / wrong_pin_pool | A legacy pin is absent, duplicated, malformed/oversized, refers to no recorded marker, or points to a pool different from configuration. |
| pin_pool_unverified / ambiguous_pin_pool | A configured marker could not be verified, or multiple physical pools match the pin; neither is reported as a definitely missing marker. |
| missing_identity / unknown_scope / invalid_school_scope | A configured/pinned/observed scope has no control identity, business rows reference an unknown school, or an invalid scope value was omitted from output. |
| unexpected_pool_rows / duplicate_school_ownership / rows_outside_recorded_pool | School rows occur outside the configured/recorded target or across multiple pools. |
| active_outbox_lease | An unpublished, nondead event has an unexpired lease; coordinate worker maintenance before any copy. |
| school_limit_exceeded | Scope enumeration was truncated; inspection is incomplete. |

This is a routing inventory, not the full upgrade dry-run or source fingerprint. Per-pool snapshots do not form one atomic cross-database snapshot. An inspection on a live system can change immediately afterward; later maintenance must freeze and reconcile the actual source. It does not inspect private objects, validate business contents or payloads, verify native migration receipts/schema signatures, confirm backups, fence old binaries, copy data, activate tenancy or enable moves. Missing/corrupt blobs can coexist with a successful SQL placement report. Resolve findings explicitly before the separate manifest/copy/verification workflow.

## PostgreSQL CLI acceptance

`TestInspectionPostgresFrozenSQLReadOnlyAndDetection` qualifies inspector SQL and read-only snapshots in isolated schemas. `TestLegacyPlacementInspectionPostgresActualCommand` separately builds and invokes the actual CLI through native PostgreSQL settings, with mixed shared/dedicated databases, recorded pins and frozen starter data. It checks deterministic clean output, unchanged source row hashes and table sets, an ownership fault, and redacted connection/configuration failures. The native PostgreSQL CI job enables both tests; a hosted result is required to claim this acceptance passed.

The CLI test requires `EDUPILOT_TEST_POSTGRES_DSN` plus the separate opt-in `EDUPILOT_TEST_POSTGRES_CREATE_DATABASES=1` and a role with `CREATEDB`. It accepts a single-host PostgreSQL URL with explicit `sslmode=disable|require|verify-ca|verify-full`; optional TLS certificate/SNI parameters are preserved through native pgx environment settings. It creates cryptographically random private databases and verifies each opened handle before populating its public schema. Cleanup drops only databases successfully created by that test. The supplied administration database's existing tables are never populated or dropped. Without the DSN/opt-in, the CLI test skips and its acceptance remains open; the schema-scoped test still runs when its DSN is available.

## Check persistent systemd masks on one native host

Before maintenance, inventory the canonical system service names for every legacy API, worker, scheduled seed/migration and file writer on each host. Record their original enablement and mask state separately. Stop those units and apply persistent system-manager masks under the site's change procedure. Retain the masks through backup, copy, verification and any rollback. Use the actual inventoried names in place of these examples:

```sh
python3 scripts/verify-systemd-writer-fence.py legacy-api.service legacy-worker.service
```

Exit 0 confirms that systemd reports every supplied service inactive, dead and persistently masked, with zero main/control PIDs, an empty control group and a matching mask under /etc/systemd/system. The command only reads state. Recheck after restart or interrupted maintenance. Restore the recorded service state only after the operator has selected and verified the authorized runtime.

This is a point-in-time check of named units. The operator must verify inventory completeness, prevent privileged changes, exclude unmanaged processes and other supervisors/hosts, drain leases, and fence database and object-store credentials separately. A mistyped name masked by the operator cannot prove that the intended writer was stopped. This check alone grants no copy or resume authority.

## Check legacy PostgreSQL writer credentials

Inventory the actual login roles used by legacy APIs, workers, jobs and document metadata writers on every control and dedicated database cluster. Connect directly to each expected primary with a separate operator identity and a trusted libpq service definition. Confirm the cluster identity independently. Record original role state, disable LOGIN for the inventoried roles and drain existing sessions under the site's maintenance procedure. Resolve outstanding prepared transactions according to their transaction coordinator's decision. Inventory and remove role memberships that would allow another identity to assume the writer role.

    PGSERVICE=maintenance-primary python3 scripts/verify-postgres-writer-fence.py legacy_api legacy_worker

The probe only reads PostgreSQL catalogs and prints a count on success. It rejects a standby, absent or login-enabled roles, surviving sessions, prepared transactions and grants of a listed role to any member. Repeat for each authoritative primary after quiescence and before resuming an interrupted upgrade. Keep the credentials fenced until the authorized runtime is ready. Independently exclude other writable identities, privileged changes, replication and external document-store access throughout maintenance. This check alone cannot authorize copy, resume or activation.

## Check a dedicated local ext4 document volume

Use this check for a legacy local document store on a dedicated Linux ext4 filesystem. Confirm the source document path, filesystem UUID and block device against the deployment inventory *before* entering maintenance. Use the original writer namespace and source volume. Checking a read-only backup only establishes the state of that backup.

Stop and drain the writers. Under the site's maintenance procedure, remount the whole filesystem read-only and retain that state through backup, copy and verification. Check the actual configured document directory and the independently inventoried device, for example:

    python3 scripts/verify-local-document-fence.py --root /srv/edupilot-volume/documents --device /dev/disk/by-uuid/APPROVED-UUID

The probe opens the directory and matches its kernel mount ID and device number. It requires both the mount and the ext4 superblock to be read-only, and rejects symlinked roots, subdirectory mounts, system-root mounts and nested mounts under the document directory. It reports one verified directory without printing the path, volume identifier or file contents. Use the existing document inventory command separately to verify the bytes and references.

Recheck each writer namespace and each configured source before resuming maintenance. Restrict privileged remounts, raw block access, alternate source paths and access from other hosts throughout the operation. This point-in-time result covers local filesystem writes only. SQLite ownership, remote object storage, other filesystems and upgrade authorization have separate gates.
