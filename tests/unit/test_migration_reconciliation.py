"""Tests for concurrent migration reconciliation."""

import hashlib
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "scripts"))

import migrate_all


class MigrationReconciliationTests(unittest.TestCase):

    def run_scenario(self, matching):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "000002_probe.sql"
            path.write_text("SELECT 1;\n")
            checksum = hashlib.sha256(path.read_bytes()).hexdigest()

            migrations = [
                (1, "core_schema", "0" * 64, path),
                (2, "probe", checksum, path),
            ]

            calls = []

            def fake_sql(query):
                calls.append(query)

                if query.startswith("SELECT to_regclass"):
                    return "t"

                if query.startswith("SELECT count(*)"):
                    return "1"

                if query.startswith("BEGIN;"):
                    raise RuntimeError("Migration state changed")

                if query.startswith("SELECT name ||"):
                    saved = checksum if matching else "f" * 64
                    return f"probe|{saved}"

                raise AssertionError("Unexpected SQL statement")

            with patch.object(migrate_all, "sql", side_effect=fake_sql):
                with patch.object(migrate_all, "validate_plan") as plan:
                    with patch.object(migrate_all, "discover_migrations", return_value=migrations):
                        if matching:
                            migrate_all.apply_migrations()
                        else:
                            with self.assertRaisesRegex(RuntimeError, "Migration state changed"):
                                migrate_all.apply_migrations()

                    self.assertEqual(plan.call_count, 3 if matching else 1)

            attempts = sum(q.startswith("BEGIN;") for q in calls)
            self.assertEqual(attempts, 1)

    def test_matching_migration_is_skipped(self):
        self.run_scenario(True)

    def test_checksum_mismatch_is_rejected(self):
        self.run_scenario(False)
