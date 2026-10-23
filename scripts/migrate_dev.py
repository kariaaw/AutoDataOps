
#!/usr/bin/env python3

"""
AutoDataOps local database migration runner.

Features:
- SQL transaction
- SHA256 checksum
- Migration version tracking
- Refusal to overwrite changed migrations
- Database integration smoke tests
"""

import hashlib
import os
import subprocess
import sys

from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]

MIGRATION = (
    ROOT
    / "infrastructure/docker/migrations"
    / "000001_core_schema.sql"
)

SMOKE = (
    ROOT
    / "tests/integration"
    / "postgres_schema_smoke.sql"
)

CONTAINER = "autodataops-dev-postgres-1"

VERSION = 1


def sql(query: str) -> str:

    target = os.environ.get("AUTODATAOPS_MIGRATION_TARGET", "main")
    if target not in ("main", "lab", "concurrency"):
        raise RuntimeError("Invalid migration target")

    databases = {
        "lab": "autodataops_migration_lab",
        "concurrency": "autodataops_concurrency_lab",
    }

    database = databases.get(target)
    options = ["-e", f"POSTGRES_DB={database}"] if database else []

    command = [
        "docker",
        "exec",
        "-i",
        *options,
        CONTAINER,
        "sh",
        "-ec",
        (
            'PGPASSWORD="$POSTGRES_PASSWORD" '
            'exec psql '
            '-h 127.0.0.1 '
            '-U "$POSTGRES_USER" '
            '-d "$POSTGRES_DB" '
            '-X -q -A -t '
            '-v ON_ERROR_STOP=1'
        ),
    ]

    result = subprocess.run(
        command,
        input=query,
        text=True,
        capture_output=True,
    )

    if result.returncode != 0:

        message = (
            result.stderr.strip()
            or f"psql failed: {result.returncode}"
        )

        raise RuntimeError(message)

    return result.stdout.strip()


def main() -> None:

    contents = MIGRATION.read_bytes()

    checksum = hashlib.sha256(
        contents
    ).hexdigest()

    existing = sql(
        "SELECT "
        "to_regclass('core.schema_migrations') "
        "IS NOT NULL;"
    )

    if existing not in ("t", "f"):

        raise RuntimeError(
            f"Unexpected migration registry: {existing!r}"
        )

    saved_checksum = ""

    if existing == "t":

        saved_checksum = sql(
            "SELECT btrim(checksum) "
            "FROM core.schema_migrations "
            "WHERE version = 1;"
        )

    if saved_checksum:

        if saved_checksum != checksum:

            raise RuntimeError(
                "Migration checksum mismatch. "
                "Database modification refused."
            )

        print(
            "[OK] Migration 000001 already applied.",
            flush=True,
        )

        print(
            "[OK] Migration checksum verified.",
            flush=True,
        )

    else:

        if existing == "t":

            raise RuntimeError(
                "Migration registry exists "
                "without version 000001. "
                "Manual inspection required."
            )

        print(
            "[INFO] Applying migration 000001...",
            flush=True,
        )

        statement = (
            "BEGIN;\n"
            "SET LOCAL lock_timeout = '10s';\n"
            "SELECT pg_advisory_xact_lock(870001);\n"
            + contents.decode("utf-8")
            + (
                "\nINSERT INTO "
                "core.schema_migrations "
                "(version, name, checksum) "
                f"VALUES ({VERSION}, "
                f"'core_schema', '{checksum}');\n"
            )
            + "COMMIT;\n"
        )

        sql(statement)

        print(
            "[OK] Migration committed.",
            flush=True,
        )

    # Verify schema

    count = sql(
        "SELECT count(*) "
        "FROM pg_tables "
        "WHERE schemaname = 'core';"
    )

    if count != "7":

        raise RuntimeError(
            f"Expected 7 tables, found {count}"
        )

    print(
        "[OK] Seven database tables verified.",
        flush=True,
    )

    # Verify rollback-only integration tests

    before = sql(
        "SELECT count(*) "
        "FROM core.organizations;"
    )

    print(
        "[INFO] Running database smoke tests...",
        flush=True,
    )

    sql(
        SMOKE.read_text(
            encoding="utf-8"
        )
    )

    print(
        "[OK] Database smoke tests passed.",
        flush=True,
    )

    after = sql(
        "SELECT count(*) "
        "FROM core.organizations;"
    )

    if before != after:

        raise RuntimeError(
            "Test rollback verification failed."
        )

    print(
        "[OK] Test data rollback verified.",
        flush=True,
    )


if __name__ == "__main__":

    try:

        main()

    except (
        OSError,
        UnicodeError,
        RuntimeError,
    ) as error:

        print(
            f"[ERROR] {error}",
            file=sys.stderr,
        )

        sys.exit(1)

