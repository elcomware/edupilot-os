# Prepare the group tenancy upgrade

The current release checkpoint can validate a proposed mapping file offline. Use the native command from the repository root:

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
