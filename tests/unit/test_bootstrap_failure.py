import hashlib
import sys
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

sys.path.insert(
    0,
    str(Path(__file__).resolve().parents[2] / "scripts")
)

import migrate_all


class BootstrapFailureTests(unittest.TestCase):

    def test_missing_registry_preserves_original_error(self):
        contents = b"SELECT 1 / 0;"
        migration_file = Mock()
        migration_file.read_bytes.return_value = contents

        migration = (
            1,
            "core_schema",
            hashlib.sha256(contents).hexdigest(),
            migration_file,
        )

        def fake_sql(query):
            if query.startswith("BEGIN;"):
                raise RuntimeError("original migration failure")

            if "to_regclass('core.schema_migrations')" in query:
                return "f"

            raise AssertionError("Unexpected SQL query")

        with patch.object(migrate_all, "validate_plan"):
            with patch.object(
                migrate_all,
                "discover_migrations",
                return_value=[migration],
            ):
                with patch.object(
                    migrate_all,
                    "sql",
                    side_effect=fake_sql,
                ):
                    with self.assertRaisesRegex(
                        RuntimeError,
                        "original migration failure",
                    ):
                        migrate_all.apply_migrations()
