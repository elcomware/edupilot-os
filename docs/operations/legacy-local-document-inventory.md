# Inventory a frozen starter document backup

`tenant-document-inventory` is an offline, read-only inspection command for the Documents module’s local private storage. It verifies the starter record envelope and its original school/UUID path, then returns a deterministic version 1 aggregate. It does not start EduPilot OS, open SQL, load a session, invent a principal, upload files, or contact S3.

Use an approved version 1 upgrade mapping manifest accepted by the existing `tenantupgrade.ReadManifest` validator. Every school directory must match a `legacySchool` in that mapping. The complete mapping is validated, but this document component digest binds only the sorted legacy-school allowlist (including empty schools), verified paths and original content. It deliberately excludes `sourceSha256`, target identities, timezone and locale: the document digest can become an input to an aggregate frozen source digest without circular dependence. Finalizing that aggregate or reallocating target UUIDs does not change this source-content digest. Syntactic validation does not approve a mapping on an operator’s behalf.

This increment is a **post-preparation verifier**, not the initial baseline generator. A real operation requires a complete, already approved manifest with the actual prepared `sourceSha256` and allocated target IDs. Do not guess those values, insert placeholder hashes/IDs, or treat syntax validation as approval. The future source-preparation phase owns establishing the first baseline and its validated input; this command does not yet accept a separate legacy-school allowlist for that cold-start step.

## Freeze the source first

Inspect an immutable, consistent backup of the **whole** local private document root, preferably mounted read-only. Keep the original records and their envelope bytes. Stop writers or take an appropriate filesystem snapshot before copying the backup. The command performs no source writes, creates no missing directories, and does not repair, delete or rename anything. Ordinary reads may update access times on a writable filesystem; a read-only mount avoids that filesystem side effect.

This command does not establish a live completeness boundary, coordinate with an API/outbox/SQL transaction, or prove that a backup already contained every historical document. A successful unpinned inventory describes the valid records present in this particular frozen root. Comparing against a separately approved digest detects omissions or changes relative to that exact baseline; calculating a new baseline after an omission cannot detect the earlier loss. S3 bucket enumeration, S3 checksums, object versions and provider evidence are outside this command.

## Build and run privately

Use the repository’s supported Go toolchain (Go 1.26.10 or newer, matching `go.mod`). Build outside the source backup:

```sh
mkdir -p ./tmp
go build -o ./tmp/tenant-document-inventory ./cmd/tenant-document-inventory
```

The root and manifest arguments are required. There is no configured/environment/default fallback for a missing root. The inspected document root must use an existing physical path without symlink components or raw `..` parent components; those inputs are rejected before path cleaning. This ancestor-path restriction concerns the document root. The manifest’s final entry must also be private, regular and not a symlink, but its parent-directory aliases are not rewritten or prohibited by this command; keep the approved manifest in a trusted private location. On the supported POSIX filesystem, the root and every nested directory must have no group/other permission bits (0700 or a read-only 0500); each record and the manifest must be private regular files (0600 or read-only 0400). Keep the manifest outside the document root, since an extra manifest file inside the root is an unexpected entry.

Create the output in a separate private working directory. The aggregate and its digest are sensitive operational evidence: do not publish them to application logs, shared terminal captures, CI logs or public tickets. The command prints no document IDs, paths, school names, source markers or content, and diagnostics do not echo supplied arguments or expected digests.

```sh
umask 077
./tmp/tenant-document-inventory \
  -manifest "$APPROVED_MAPPING_JSON" \
  -root "$FROZEN_PRIVATE_DOCUMENT_ROOT" > "$PRIVATE_INVENTORY_JSON"
```

Only retain the output if the exit status is zero. Bind the approved full mapping separately when assembling the overall upgrade evidence; this component digest is not a mapping fingerprint. Validation finishes before any summary is written; rejected input produces no aggregate. An output-device failure can still interrupt the final write, so an exit failure always invalidates the destination. The tool does not create or chmod the destination: the operator’s private directory, umask and redirection protect it. Avoid overwriting the only approved baseline.

To check the same frozen backup and mapping against an independently retained baseline, supply its approved digest:

```sh
./tmp/tenant-document-inventory \
  -manifest "$APPROVED_MAPPING_JSON" \
  -root "$FROZEN_PRIVATE_DOCUMENT_ROOT" \
  -expected-sha256 "$APPROVED_DOCUMENT_INVENTORY_SHA256" > "$PRIVATE_CHECK_JSON"
```

Expected digests must be exactly 64 lowercase hexadecimal characters. Explicitly supplying an empty value (including an unset shell variable) is an error; only omitting the flag opts out of comparison. A mismatch fails without printing either digest or a partial summary. Treat a failure as a reason to review the private backup, mapping, shape and limits; do not remove unexpected files or recalculate the baseline simply to make the comparison pass.

## Accepted layout and limits

The only nonempty layout is:

```text
DOCUMENT_ROOT/
  schools/
    APPROVED_LEGACY_SCHOOL/
      documents/
        CANONICAL_UUID_V4
```

The root may be completely empty. `schools/` may be empty, and an approved school’s `documents/` may be empty. If a school directory exists, it must contain exactly its `documents/` directory. No other root or school entries are allowed. Every entry under `documents/` must be a canonical lowercase random UUID v4 regular file with a valid `EDUPILOT-DOC/1` envelope bound to that exact original path. Raw parent-path components in the root argument, symlinks (including a symlinked root or ancestor), FIFOs, sockets, directories used as records, unknown/foreign school directories, orphan files, temporary `.upload-*` files and other unexpected entries are rejected rather than ignored.

Each record uses the existing application read bound (8 MiB of content plus the envelope), checksum and media checks used by authenticated downloads: supported PDF, PNG, JPEG or valid UTF-8 plain text. These checks detect damaged or misplaced bytes; they are not malware scanning or a cryptographic claim against a writer who can replace both content and its envelope.

| Bound | Maximum |
| --- | ---: |
| Approved mappings | Existing manifest limit: 4,096 schools / 4 MiB JSON |
| All entries read across the entire directory tree | 20,000 |
| Verified document records | 10,000 |
| Total original content bytes | 1 GiB (1,073,741,824 bytes) |
| Original content in one record | 8 MiB |

Enumeration reads at most the remaining entry budget plus one sentinel entry, sorts each bounded directory lexicographically, and opens one document at a time. A limit breach rejects the whole inventory. Larger backups need a separately reviewed tool/limit change; this version does not silently truncate or offer a skip-invalid switch.

## Version 1 aggregate and digest

The private JSON result contains exactly these fields:

| Field | Meaning |
| --- | --- |
| `version` | Inventory/hash format version, currently `1` |
| `schools` | Count of approved mapping entries, including schools with no documents |
| `documents` | Count of verified records actually present |
| `contentBytes` | Sum of original content bytes, excluding envelope overhead |
| `sha256` | Lowercase SHA-256 of the complete canonical inventory framing |

A valid empty existing root returns zero documents and zero content bytes with a nonempty deterministic digest. A missing root fails. The digest does not depend on the absolute backup location, directory listing order, file timestamps or permissions; those permissions are still validated. Incidental empty approved directory scaffolding does not change the document aggregate. Changing approved legacy-school membership changes it, even if the added or removed school has zero files. Reordering equivalent mappings or changing only `sourceSha256` or target identity/configuration fields does not change it.

The hash input uses these exact steps:

1. Frame the UTF-8 domain string `EDUPILOT-LOCAL-DOCUMENT-INVENTORY/1`, then the UTF-8 string `schools`. Append the approved school count as an unsigned 64-bit big-endian integer, then frame each legacy-school slug in bytewise lexical order, including schools without documents.
2. For each verified record in school-name/UUID bytewise lexical order, frame the UTF-8 string `document`, its verified relative storage path `schools/{legacySchool}/documents/{uuid}`, and the complete original content bytes.
3. Frame the UTF-8 string `totals`, then append the approved school count, document count and content-byte count as unsigned 64-bit big-endian integers.

Every **frame** is an unsigned 64-bit big-endian byte length followed by exactly that many bytes. Framing prevents concatenation ambiguity. A different envelope/media/path policy or hash framing requires a new inventory format version. This digest is a snapshot comparison aid, not a signature or proof of authority.
