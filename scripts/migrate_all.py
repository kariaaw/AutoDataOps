"""Transactional, versioned database migration runner."""

import hashlib
import sys

from migration_catalog import discover_migrations
from migration_plan import main as validate_plan
from migrate_dev import sql


def apply_migrations():
    validate_plan()
    migrations = discover_migrations()

    exists = sql(
        "SELECT to_regclass('core.schema_migrations') "
        "IS NOT NULL;"
    )

    if exists not in ("t", "f"):
        raise RuntimeError("Invalid registry state")

    applied = 0

    if exists == "t":
        applied = int(sql(
            "SELECT count(*) FROM core.schema_migrations;"
        ))

    for version, name, checksum, path in migrations[applied:]:
        contents = path.read_bytes()

        if hashlib.sha256(contents).hexdigest() != checksum:
            raise RuntimeError("Migration file changed")

        if version == 1:
            guard = (
                "DO $$ BEGIN "
                "IF to_regclass('core.schema_migrations') IS NOT NULL "
                "THEN RAISE EXCEPTION 'Registry already exists'; "
                "END IF; END $$;"
            )
        else:
            previous = migrations[version - 2][2]
            guard = (
                "DO $$ BEGIN "
                "IF (SELECT count(*) FROM core.schema_migrations) "
                f"<> {version - 1} OR NOT EXISTS ("
                "SELECT 1 FROM core.schema_migrations "
                f"WHERE version = {version - 1} "
                f"AND btrim(checksum) = '{previous}') "
                "THEN RAISE EXCEPTION 'Migration state changed'; "
                "END IF; END $$;"
            )

        script = (
            "BEGIN;\n"
            "SET LOCAL lock_timeout = '10s';\n"
            "SELECT pg_advisory_xact_lock(870001);\n"
            + guard + "\n"
            + contents.decode("utf-8") + "\n"
            + "INSERT INTO core.schema_migrations "
            + "(version, name, checksum) "
            + f"VALUES ({version}, '{name}', '{checksum}');\n"
            + "COMMIT;\n"
        )

        print(f"[APPLYING] {version:06d}_{name}", flush=True)
        try:
            sql(script)
        except RuntimeError:
            registry_state = sql(
                "SELECT to_regclass('core.schema_migrations') "
                "IS NOT NULL;"
            )

            if registry_state != "t":
                raise

            saved = sql(
                "SELECT name || '|' || btrim(checksum) "
                "FROM core.schema_migrations "
                f"WHERE version = {version};"
            )

            if saved != f"{name}|{checksum}":
                raise

            validate_plan()
            print(f"[SKIPPED] {version:06d}_{name}", flush=True)
            continue

        print(f"[COMMITTED] {version:06d}_{name}", flush=True)

    validate_plan()
    print("[PASS] Migration execution completed")


def main():
    if sys.argv[1:] == ["--plan"]:
        validate_plan()
        return

    if sys.argv[1:] == ["--apply"]:
        import os

        target = os.environ.get("AUTODATAOPS_MIGRATION_TARGET")

        allowed = {
            "lab": "autodataops_migration_lab",
            "concurrency": "autodataops_concurrency_lab",
        }

        if target not in allowed:
            raise RuntimeError("Explicit test database target required")

        actual = sql("SELECT current_database();")

        if actual != allowed[target]:
            raise RuntimeError("Migration target verification failed")

        import fcntl
        from pathlib import Path

        lock_dir = Path.home() / ".local/state/autodataops"
        lock_dir.mkdir(parents=True, exist_ok=True)
        lock_dir.chmod(0o700)

        lock_path = lock_dir / "migration-lab.lock"

        with lock_path.open("a+") as lock:
            lock_path.chmod(0o600)

            try:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError:
                raise RuntimeError("Another local migration is running")

            apply_migrations()

        return

    raise RuntimeError("Use --plan or --apply")


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, UnicodeError) as error:
        print(f"[ERROR] {error}", file=sys.stderr)
        sys.exit(1)
