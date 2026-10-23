"""Migration discovery and integrity metadata."""

import hashlib
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DIRECTORY = ROOT / "infrastructure/docker/migrations"

def discover_migrations():
    paths = sorted(DIRECTORY.glob("*.sql"))

    if not paths:
        raise RuntimeError("No SQL migrations found")

    migrations = []

    for expected, path in enumerate(paths, start=1):
        match = re.fullmatch(
            r"([0-9]{6})_([a-z][a-z0-9_]*)\.sql",
            path.name,
        )

        if match is None:
            raise RuntimeError(f"Invalid filename: {path.name}")

        version = int(match.group(1))

        if version != expected:
            raise RuntimeError("Migration version gap or duplicate")

        if path.is_symlink() or not path.is_file():
            raise RuntimeError(f"Invalid migration file: {path.name}")

        checksum = hashlib.sha256(path.read_bytes()).hexdigest()
        migrations.append((version, match.group(2), checksum, path))

    return migrations

if __name__ == "__main__":
    for version, name, checksum, path in discover_migrations():
        print(f"[OK] Migration {version:06d}: {name}")
        print(f"[SHA256] {checksum}")
    print("[PASS] Migration catalog validated")
