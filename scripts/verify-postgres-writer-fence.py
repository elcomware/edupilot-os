#!/usr/bin/env python3
"""Inspect declared legacy login roles on one PostgreSQL primary."""
import argparse
import json
import subprocess
import sys

# Use literal-quoted psql substitution. Never emit the role list or diagnostics.
CHECK = """
SET search_path = pg_catalog, pg_temp;
WITH declared AS (
    SELECT json_array_elements_text(:'writers'::json) AS name
)
SELECT NOT pg_is_in_recovery() AND NOT EXISTS (
    SELECT 1 FROM declared d LEFT JOIN pg_roles r ON r.rolname::text = d.name
    WHERE r.oid IS NULL OR r.rolcanlogin
       OR EXISTS (SELECT FROM pg_stat_activity a WHERE a.usesysid = r.oid)
       OR EXISTS (SELECT FROM pg_prepared_xacts x WHERE x.owner = r.rolname)
       OR EXISTS (SELECT FROM pg_auth_members m WHERE m.roleid = r.oid)
);
"""


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("roles", nargs="+", help="all legacy writer logins on this cluster")
    roles = parser.parse_args().roles
    if len(set(roles)) != len(roles) or any(not role for role in roles):
        parser.error("supply unique nonempty roles")
    try:
        result = subprocess.run(
            ["psql", "-X", "-qAtw", "-v", "ON_ERROR_STOP=1", "-v",
             "writers=" + json.dumps(roles)], input=CHECK, capture_output=True,
            text=True, timeout=15, check=True,
        )
        if result.stdout.strip() != "t":
            raise ValueError()
    except (OSError, subprocess.SubprocessError, ValueError):
        print("Declared PostgreSQL writer fence could not be verified.", file=sys.stderr)
        return 1
    print(json.dumps({"classification": "postgres-writer-check", "verifiedRoles": len(roles)}))
    return 0


if __name__ == "__main__":
    sys.exit(main())
