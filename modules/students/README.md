# Student records

Students owns a school's roster. It keeps one authoritative record per student,
including its names, lifecycle status, and version. Enrollment, attendance, and
guardianship belong to their own modules. Use the public
`contracts.Directory.Lookup(ctx, school, id)` to look up a student: the caller
supplies its verified school and must explicitly check `Status == "active"`
when inactive students are ineligible.

## HTTP contract

All routes are relative to `/api/v1/students`. Authenticate through the shared
session middleware. The school comes from the validated principal, never a URL
parameter or a submitted JSON field.

| Method / path | Input | Success |
| --- | --- | --- |
| GET / | `limit` (default 25, range 1–100), `offset` (default 0) | 200, `{"items":[...],"total":3}`. Only active students count; order is family name, given name, ID. |
| POST / | `{"givenName":"Ana","familyName":"Liu"}` | 201 and the created record; `Location` points at it. |
| GET /:id | — | 200 and the current record, including archived records. |
| PATCH /:id | Both names and the last observed `version` | 200 and the saved record with the next version. |
| DELETE /:id | `?version=3` | 204; archives the record and increments its version. |

Collection requests use the exact URL `/api/v1/students` (without a trailing
slash). Reads require `students.read`; writes require `students.write`. IDs
are opaque strings. Both names accept 1 to 100 Unicode code points after
trimming and reject control characters. A resource looks like:

```json
{"id":"b51abbb0189cce7baf8cb218969e5acf","givenName":"Ana","familyName":"Liu","version":1,"status":"active"}
```

An unknown ID or a student from another school gives the same 404. A stale
version or attempted edit to an archived student gives 409. On a conflict,
reload by ID and let the user reconcile their edits. Invalid names or version
values return 422; malformed JSON, unknown body fields, and malformed integer
parameters return 400. Error responses have the shape
`{"error":{"code":"version_conflict","message":"..."}}`.

## Persistence and integration

`providers.New` builds the application service with the SQL repository and
exposes its HTTP handler and public directory. The repository first resolves
the verified school's Goravel connection, then uses bound parameters and
explicit school predicates on shared and dedicated databases alike. The
database key is `(school_id, id)`; your integration should not assume that IDs
alone establish ownership.

Every successful mutation writes the new aggregate version and exactly one
outbox row in the same SQL transaction. Event types are
`students.registered.v1`, `students.updated.v1`, and `students.archived.v1`.
Their envelope carries the tenant and aggregate ID. Payloads carry
`studentId`, `version`, `status`, and `actorId`; they omit student names. A
worker may deliver an event more than once; consumers should deduplicate using
the event ID. Optimistic updates compare the expected version in the SQL
write, so two callers reading the same version cannot both commit.

The module's migrations own `students_records` and its roster index. The
application registry registers them for each physical tenant database. Add
future schema changes with a new migration ID; do not rewrite a migration
already applied in a deployment.

## Extending the reference module

Put lifecycle rules in `domain`, command/query orchestration in `application`,
and storage and HTTP mapping in their respective adapters. Other modules
import only `contracts`. Add coverage at the narrowest useful seam: domain
invariants, orchestration with a fake repository, route behavior with a trusted
test principal, or the actual SQL transaction. The repository suite runs on
SQLite and also on PostgreSQL when `EDUPILOT_TEST_POSTGRES_DSN` is configured
to create a temporary schema in a disposable database.
