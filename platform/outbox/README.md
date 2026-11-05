# Transactional outbox

Start the worker after running `go run ./cmd/migrate`, using the same connection configuration as the API:

```sh
go run ./cmd/worker
```

A repository calls `Append` inside the transaction that changes the aggregate. Rolling that transaction back also rolls back the event. The default subscriber writes metadata to `edu_audit_inbox` on the originating pool; it does not forward student payloads to an external service. The worker visits the shared and each configured dedicated pool. Independent worker processes may run concurrently. It claims one record with a 30-second lease, then commits the audit effect and the outbox acknowledgement in the same transaction. The inbox's primary key is the event ID.

On a handled failure, retry starts after one second and doubles; the eighth failed attempt marks the row dead. A process that stops with an active lease leaves the message for a worker to reclaim after expiry. Operators can inspect backlog and dead letters on **each** configured pool:

```sql
SELECT tenant_id, count(*) FROM edu_outbox
WHERE published_at IS NULL AND dead_at IS NULL GROUP BY tenant_id;

SELECT event_id, tenant_id, event_type, attempts, dead_at, last_error
FROM edu_outbox WHERE dead_at IS NOT NULL ORDER BY dead_at;
```

After fixing the cause, use a bound event ID and tenant ID to reset that dead row only. This example uses SQLite parameter syntax; use `$1` and `$2` for PostgreSQL:

```sql
UPDATE edu_outbox
SET dead_at = NULL, last_error = NULL, attempts = 0, available_at = 0,
    lease_id = NULL, lease_until = NULL
WHERE event_id = ? AND tenant_id = ? AND dead_at IS NOT NULL AND published_at IS NULL;
```

A new subscriber that performs an external effect needs its own inbox and idempotent delivery contract. The audit inbox only guarantees one local audit record per event. Events may be retried or delivered out of order, and no retention policy deletes delivered or dead records automatically.
