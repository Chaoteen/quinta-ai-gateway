import importlib.util
from contextlib import closing
import json
from pathlib import Path
import sqlite3
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('audit', Path(__file__).with_name('export_security_audit.py'))
audit = importlib.util.module_from_spec(spec)
spec.loader.exec_module(audit)


class AuditExportTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.db = Path(self.temp.name) / 'source.db'
        self.out = Path(self.temp.name) / 'logs'
        with closing(sqlite3.connect(self.db)) as db, db:
            db.execute('CREATE TABLE credential_reset_audits(id INTEGER PRIMARY KEY, actor_id INT, target_id INT, target_tenant_id INT, credential_type TEXT, created_at INT, secret TEXT)')
            db.execute('CREATE TABLE supplier_audits(id INTEGER PRIMARY KEY, actor_id INT, action TEXT, resource_id INT, created_at INT)')
            db.execute("INSERT INTO credential_reset_audits VALUES(1, 1, 2, 1, 'totp', 100, 'never-export-this')")
            db.execute("INSERT INTO supplier_audits VALUES(1, 1, 'supplier.create', 1, 101)")

    def records(self):
        return [json.loads(line) for p in self.out.glob('*.jsonl') for line in p.read_text().splitlines()]

    def test_restart_redaction_and_read_only_source(self):
        before = self.db.read_bytes()
        self.assertEqual(audit.export_once(self.db, self.out), 2)
        self.assertEqual(audit.export_once(self.db, self.out), 0)
        self.assertEqual(before, self.db.read_bytes())
        records = self.records()
        self.assertEqual(len(records), 4)
        self.assertNotIn('never-export-this', json.dumps(records))
        self.assertTrue(all(not row['remote_delivery_verified'] for row in records if row['event_type'] == 'exporter_heartbeat'))

    def test_checkpoint_failure_replays_stable_ids(self):
        audit.export_once(self.db, self.out, limit=0)
        with patch.object(audit, 'atomic_json', side_effect=OSError('disk failed')):
            with self.assertRaises(OSError):
                audit.export_once(self.db, self.out)
        audit.export_once(self.db, self.out)
        rows = [row for row in self.records() if 'event_id' in row]
        self.assertEqual([row['event_id'] for row in rows[:2]], [row['event_id'] for row in rows[2:]])

    def test_output_failure_does_not_advance(self):
        audit.export_once(self.db, self.out, limit=0)
        original = (self.out / 'checkpoint.json').read_bytes()
        with patch.object(audit.os, 'fsync', side_effect=OSError('disk full')):
            with self.assertRaises(OSError):
                audit.export_once(self.db, self.out)
        self.assertEqual(original, (self.out / 'checkpoint.json').read_bytes())

    def test_restore_rewind_fails_closed(self):
        audit.export_once(self.db, self.out)
        with closing(sqlite3.connect(self.db)) as db, db:
            db.execute('DELETE FROM supplier_audits')
        with self.assertRaisesRegex(ValueError, 'rewound'):
            audit.export_once(self.db, self.out)

    def test_uncommitted_rows_not_exported(self):
        with closing(sqlite3.connect(self.db)) as db, db:
            db.execute("INSERT INTO supplier_audits VALUES(2, 1, 'uncommitted', 2, 102)")
            self.assertEqual(audit.export_once(self.db, self.out), 2)
            db.rollback()
        self.assertNotIn('uncommitted', json.dumps(self.records()))

    def test_killed_partial_write_is_recovered(self):
        audit.export_once(self.db, self.out, limit=0)
        path = next(self.out.glob('*.jsonl'))
        with path.open('ab') as stream:
            stream.write(b'{"event_id":"partial')
        self.assertEqual(audit.export_once(self.db, self.out), 2)
        self.assertEqual(len(self.records()), 4)


if __name__ == '__main__':
    unittest.main()
