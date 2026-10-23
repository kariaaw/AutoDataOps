"""Unit tests for SQL migration discovery."""

import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))

import migration_catalog


class MigrationCatalogTests(unittest.TestCase):

    def discover(self, *names):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            for name in names:
                (directory / name).write_text("SELECT 1;\n")

            with patch.object(migration_catalog, "DIRECTORY", directory):
                return migration_catalog.discover_migrations()

    def test_single_migration(self):
        result = self.discover("000001_core_schema.sql")
        self.assertEqual(len(result), 1)
        self.assertEqual(result[0][0], 1)
        self.assertEqual(result[0][1], "core_schema")

    def test_version_order(self):
        result = self.discover(
            "000002_security.sql",
            "000001_core_schema.sql",
        )
        self.assertEqual([x[0] for x in result], [1, 2])

    def test_missing_version(self):
        with self.assertRaises(RuntimeError):
            self.discover(
                "000001_core_schema.sql",
                "000003_security.sql",
            )

    def test_duplicate_version(self):
        with self.assertRaises(RuntimeError):
            self.discover(
                "000001_first.sql",
                "000001_second.sql",
            )

    def test_invalid_filename(self):
        with self.assertRaises(RuntimeError):
            self.discover("migration.sql")

    def test_empty_directory(self):
        with self.assertRaises(RuntimeError):
            self.discover()


if __name__ == "__main__":
    unittest.main()
