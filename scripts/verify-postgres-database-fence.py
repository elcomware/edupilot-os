#!/usr/bin/env python3
"""Check closed legacy databases from an independent database on the primary."""
import argparse
import json
import subprocess
import sys

CHECK = """
SET search_path = pg_catalog, pg_temp;
WITH declared AS (
    SELECT json_array_elements_text(:'databases'::json) AS name
)
SELECT NOT pg_is_in_recovery()
   AND (SELECT system_identifier::text FROM pg_control_system()) = :'cluster'
   AND NOT EXISTS (
       SELECT FROM declared d LEFT JOIN pg_database db ON db.datname::text = d.name
       WHERE db.oid IS NULL OR db.datallowconn
          OR EXISTS (SELECT FROM pg_stat_activity a WHERE a.datid = db.oid)
          OR EXISTS (SELECT FROM pg_prepared_xacts x WHERE x.database = db.datname)
   );
"""


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--cluster-id", required=True)
    parser.add_argument("databases", nargs="+")
    args = parser.parse_args()
    if len(set(args.databases)) != len(args.databases) or not all(args.databases):
        parser.error("supply unique nonempty databases")
    try:
        result = subprocess.run(
            ["psql", "-X", "-qAtw", "-v", "ON_ERROR_STOP=1", "-v",
             "databases=" + json.dumps(args.databases), "-v", "cluster=" + args.cluster_id],
            input=CHECK, capture_output=True, text=True, timeout=15, check=True,
        )
    except (OSError, subprocess.SubprocessError):
        result = None
    if result is None or result.stdout.strip() != "t":
        print("PostgreSQL database fence could not be verified.", file=sys.stderr)
        return 1
    print(json.dumps({"classification": "postgres-database-fence", "verifiedDatabases": len(args.databases)}))
    return 0


if __name__ == "__main__":
    sys.exit(main())
