"""Unit tests for the migration execution path."""

import hashlib
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "scripts"))
import migrate_all


class MigrationRunnerTests(unittest.TestCase):

    def test_no_pending_migrations(self):
        queries = []

        def fake_sql(query):
            queries.append(query)
            if "to_regclass" in query:
                return "t"
            if "count(*)" in query:
                return "1"
            raise AssertionError("Unexpected SQL execution")

        migrations = [(1, "core_schema", "0" * 64, None)]

        with patch.object(migrate_all, "sql", side_effect=fake_sql), \
             patch.object(migrate_all, "validate_plan"), \
             patch.object(migrate_all, "discover_migrations", return_value=migrations):
            migrate_all.apply_migrations()

        self.assertEqual(len(queries), 2)

    def test_new_migration_transaction(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "000002_test.sql"
            path.write_text("SELECT 1;\n", encoding="utf-8")
            checksum = hashlib.sha256(path.read_bytes()).hexdigest()

            migrations = [
                (1, "core_schema", "0" * 64, path),
                (2, "test", checksum, path),
            ]

            queries = []

            def fake_sql(query):
                queries.append(query)
                if "to_regclass" in query:
                    return "t"
                if "count(*)" in query:
                    return "1"
                return ""

            with patch.object(migrate_all, "sql", side_effect=fake_sql), \
                 patch.object(migrate_all, "validate_plan"), \
                 patch.object(migrate_all, "discover_migrations", return_value=migrations):
                migrate_all.apply_migrations()

            statement = queries[-1]
            self.assertTrue(statement.startswith("BEGIN;"))
            self.assertIn("pg_advisory_xact_lock", statement)
            self.assertIn("INSERT INTO core.schema_migrations", statement)
            self.assertTrue(statement.rstrip().endswith("COMMIT;"))


if __name__ == "__main__":
    unittest.main()
