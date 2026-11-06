# Prepare the group tenancy upgrade

The current release checkpoint can validate a proposed mapping file offline and inspect legacy SQL placement using read-only connections. Use the native command from the repository root:

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

A successful validation covers the submitted file. Database inspection, private-object reconciliation, a verified backup, maintenance fencing, copy with per-pool receipts and verification against the frozen source are separate requirements. Operators should keep the current runtime in school scope. The [release board](../release/status.md) records which upgrade phases have passed.


## Inspect legacy SQL placement

From the native deployment directory, with the same environment and `.env` used by the application:

```sh
go run ./cmd/tenant-upgrade -mode=inspect-placement > /private/placement-inspection.json
```

The mode must be explicit. A blank/default mode refuses to run. `inspect-placement` takes no manifest; `validate-manifest -manifest=PATH` remains an offline file validator. Native APP_KEY/configuration validation and process-environment-over-`.env` precedence are preserved. This command does not boot HTTP, run migrations, dispatch events, provision routes or create placement pins. It reads DB_CONNECTION/DB_DATABASE and the native PostgreSQL settings plus EDUPILOT_DEDICATED_DATABASES, using the application's validated connection names and legacy owner strings. PostgreSQL uses the same native public search_path; no arbitrary DSN/SQL input is accepted.

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
