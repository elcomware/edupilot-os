#!/usr/bin/env bash
# Only a new cluster on a private Unix socket. Ignore inherited service/DSN settings.
set -euo pipefail
pgbin=${1:?supply PostgreSQL bin directory}
dir=$(mktemp -d)
export PATH="$pgbin:$PATH"
pg() { env -i PATH="$PATH" LC_ALL=C LD_LIBRARY_PATH="${LD_LIBRARY_PATH:-}" PGHOST="$dir" PGPORT=5432 PGDATABASE=postgres PGUSER=postgres "$@"; }
cleanup() {
  if [[ -f "$dir/data/postmaster.pid" ]]; then
    pg pg_ctl -D "$dir/data" -m immediate -w stop >/dev/null || return 1
  fi
  rm -rf "$dir"
}
trap cleanup EXIT
pg initdb -D "$dir/data" -U postgres -A trust >/dev/null
pg pg_ctl -D "$dir/data" -l "$dir/server.log" -o "-k '$dir' -c listen_addresses='' -c max_prepared_transactions=4" -w start >/dev/null || { cat "$dir/server.log" >&2; exit 1; }
sql() { pg psql -XqAtw -v ON_ERROR_STOP=1 -c "$1"; }
check() { pg python3 scripts/verify-postgres-writer-fence.py legacy_api legacy_worker; }
reject() { if check; then echo 'Unexpected fence acceptance' >&2; exit 1; fi; }
reject
sql 'CREATE ROLE legacy_api LOGIN; CREATE ROLE legacy_worker NOLOGIN'
reject
pg env PGUSER=legacy_api PGAPPNAME=fence_drill psql -XqAtw -c 'SELECT pg_sleep(120)' >/dev/null 2>&1 &
session=$!
for i in {1..100}; do
  [[ $(sql "SELECT count(*) FROM pg_stat_activity WHERE application_name='fence_drill' AND state='active'") == 1 ]] && break
  sleep .1
done
[[ $(sql "SELECT count(*) FROM pg_stat_activity WHERE application_name='fence_drill' AND state='active'") == 1 ]]
sql 'ALTER ROLE legacy_api NOLOGIN'
reject
if pg env PGUSER=legacy_api psql -XqAtw -c 'SELECT 1' >/dev/null 2>&1; then exit 1; fi
sql "SELECT pg_terminate_backend(pid, 5000) FROM pg_stat_activity WHERE application_name='fence_drill'" >/dev/null
wait "$session" || true
check
sql 'CREATE ROLE delegate_role LOGIN; GRANT legacy_worker TO delegate_role'
reject
sql 'REVOKE legacy_worker FROM delegate_role; ALTER ROLE legacy_worker LOGIN'
pg env PGUSER=legacy_worker psql -XqAtw -v ON_ERROR_STOP=1 -c "BEGIN; SELECT 1; PREPARE TRANSACTION 'fence_drill'"
sql 'ALTER ROLE legacy_worker NOLOGIN'
reject
sql "ROLLBACK PREPARED 'fence_drill'"
check

# Database-wide exclusion is checked from the maintenance database, including
# connections whose login does not appear in the legacy writer inventory.
cluster=$(sql 'SELECT system_identifier FROM pg_control_system()')
dbcheck() { pg python3 scripts/verify-postgres-database-fence.py --cluster-id "$@"; }
must_fail() { if "$@"; then echo 'Unexpected database fence acceptance' >&2; exit 1; fi; }
sql 'CREATE DATABASE source_shared'
sql 'CREATE DATABASE source_dedicated'
pg env PGDATABASE=source_dedicated psql -XqAtw -v ON_ERROR_STOP=1 -c "BEGIN; SELECT 1; PREPARE TRANSACTION 'pending_copy'"
pg env PGDATABASE=source_shared PGUSER=delegate_role PGAPPNAME=unknown_writer psql -XqAtw -c 'SELECT pg_sleep(120)' >/dev/null 2>&1 &
other=$!
for i in {1..100}; do
  [[ $(sql "SELECT count(*) FROM pg_stat_activity WHERE application_name='unknown_writer'") == 1 ]] && break
  sleep .1
done
[[ $(sql "SELECT count(*) FROM pg_stat_activity WHERE application_name='unknown_writer'") == 1 ]]
sql 'ALTER DATABASE source_shared ALLOW_CONNECTIONS false'
sql 'ALTER DATABASE source_dedicated ALLOW_CONNECTIONS false'
must_fail dbcheck "$cluster" source_shared
must_fail dbcheck "$cluster" source_dedicated
if pg env PGDATABASE=source_shared PGUSER=delegate_role psql -XqAtw -c 'SELECT 1' >/dev/null 2>&1; then exit 1; fi
sql "SELECT pg_terminate_backend(pid, 5000) FROM pg_stat_activity WHERE application_name='unknown_writer'" >/dev/null
wait "$other" || true
dbcheck "$cluster" source_shared
sql 'ALTER DATABASE source_dedicated ALLOW_CONNECTIONS true'
pg env PGDATABASE=source_dedicated psql -XqAtw -v ON_ERROR_STOP=1 -c "ROLLBACK PREPARED 'pending_copy'"
must_fail dbcheck "$cluster" source_shared source_dedicated
sql 'ALTER DATABASE source_dedicated ALLOW_CONNECTIONS false'
dbcheck "$cluster" source_shared source_dedicated
must_fail dbcheck wrong_cluster source_shared source_dedicated
must_fail dbcheck "$cluster" source_shared missing_database
