"""Read-only migration planning and integrity checks."""

import sys

from migration_catalog import discover_migrations
from migrate_dev import sql


def main():
    migrations = discover_migrations()

    registry = sql(
        "SELECT to_regclass('core.schema_migrations') "
        "IS NOT NULL;"
    )

    if registry == "t":
        result = sql(
            "SELECT version, name, btrim(checksum) "
            "FROM core.schema_migrations ORDER BY version;"
        )
        applied = [r.split("|") for r in result.splitlines() if r]
        if not applied:
            raise RuntimeError("Migration registry is empty")

    elif registry == "f":
        count = sql(
            "SELECT count(*) FROM pg_tables "
            "WHERE schemaname = 'core';"
        )
        if count != "0":
            raise RuntimeError("Untracked core tables detected")
        applied = []

    else:
        raise RuntimeError("Unexpected registry status")

    if len(applied) > len(migrations):
        raise RuntimeError("Unknown applied migrations")

    for index, row in enumerate(applied):
        if len(row) != 3:
            raise RuntimeError("Invalid migration registry row")

        version, name, checksum, _ = migrations[index]

        if row != [str(version), name, checksum]:
            raise RuntimeError(
                f"Migration {version:06d} integrity mismatch"
            )

        print(f"[VERIFIED] {version:06d}_{name}")

    pending = migrations[len(applied):]

    for version, name, _, _ in pending:
        print(f"[PENDING] {version:06d}_{name}")

    print(f"[INFO] Applied: {len(applied)}")
    print(f"[INFO] Pending: {len(pending)}")
    print("[PASS] Migration plan validated")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError) as error:
        print(f"[ERROR] {error}", file=sys.stderr)
        sys.exit(1)
