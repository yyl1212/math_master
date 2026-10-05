"""真实隔离 PostgreSQL 的备份/恢复安全契约。"""
from datetime import datetime
import errno
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import threading
import time
import unittest
import uuid
from unittest.mock import patch

OPS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(OPS))
import common

path = OPS / 'database-snapshot.py'
snapshot = None
if path.exists():
    spec = importlib.util.spec_from_file_location('database_snapshot', path)
    snapshot = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(snapshot)

IMAGE = 'postgres:17.11@sha256:e31e3d5327d1806f6177827c9710643e4f35f7ab3f14d26d05332753d3e95ee0'
REVISION = '2' * 40


class SnapshotTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory(dir='/private/tmp' if Path('/private/tmp').exists() else None)
        cls.root = Path(cls.temp.name)
        cls.resources = []
        cls.addClassCleanup(cls.cleanup_resources)
        if snapshot is None:
            return
        result = common.docker(['image', 'inspect', IMAGE])
        if result.returncode:
            result = common.docker(['pull', '--platform', 'linux/amd64', IMAGE])
            if result.returncode:
                raise RuntimeError('isolated PostgreSQL image unavailable')
        cls.container, cls.database = cls.new_database()
        snapshot.sql(cls.container, cls.database, 'math_master_preview',
                     "CREATE TABLE items(id bigserial PRIMARY KEY,body text NOT NULL); INSERT INTO items(body) SELECT repeat('snapshot-fixture',50) FROM generate_series(1,1000);")

    @classmethod
    def new_database(cls):
        token = uuid.uuid4().hex
        name = 'math-master-snapshot-test-' + token
        database = 'math_master_restore_' + token
        env = cls.root / (token + '.env')
        env.write_text('POSTGRES_USER=math_master_preview\nPOSTGRES_PASSWORD=' + 'b'*64 + '\nPOSTGRES_DB=' + database + '\nPOSTGRES_INITDB_ARGS=--encoding=UTF8 --lc-collate=en_US.utf8 --lc-ctype=en_US.utf8\n')
        env.chmod(0o600)
        result = common.docker(['run', '--detach', '--platform', 'linux/amd64', '--network', 'none', '--name', name, '--label', 'math-master.restore-drill=' + token, '--env-file', str(env), IMAGE])
        if result.returncode:
            raise RuntimeError('isolated PostgreSQL fixture did not start')
        cls.resources.append(name)
        for _ in range(60):
            probe = common.docker(['exec', name, 'pg_isready', '-h', '127.0.0.1', '-U', 'math_master_preview', '-d', database])
            if probe.returncode == 0:
                return name, database
            time.sleep(.2)
        raise RuntimeError('isolated PostgreSQL fixture readiness failed')

    @classmethod
    def cleanup_resources(cls):
        for resource in cls.resources:
            common.docker(['rm', '-f', '-v', resource])
        cls.temp.cleanup()

    def implementation(self):
        self.assertIsNotNone(snapshot, 'consistent snapshot tooling is missing')
        return snapshot

    def backup(self, name):
        module = self.implementation()
        out = self.root / (name + '-' + uuid.uuid4().hex)
        module.capture(self.container, self.database, 'math_master_preview', out, REVISION)
        return out

    def test_snapshot_stays_consistent_during_write(self):
        module = self.implementation()
        stop = threading.Event()
        errors = []
        def writer():
            while not stop.is_set():
                try:
                    module.sql(self.container, self.database, 'math_master_preview', "INSERT INTO items(body) VALUES('concurrent-write');")
                except Exception:
                    errors.append(True)
                    break
        thread = threading.Thread(target=writer)
        thread.start()
        try:
            out = self.backup('concurrent')
        finally:
            stop.set()
            thread.join(timeout=10)
        self.assertFalse(thread.is_alive())
        self.assertFalse(errors)
        manifest = json.loads((out/'manifest.json').read_text())
        # 确保源码快照之后还有提交；真实恢复必须仍与导出快照的摘要一致。
        module.sql(self.container, self.database, 'math_master_preview', "INSERT INTO items(body) VALUES('after-snapshot');")
        current = int(module.sql(self.container, self.database, 'math_master_preview', 'SELECT count(*) FROM items;'))
        self.assertGreater(current, manifest['tables']['items']['rows'])
        result = module.drill(out)
        self.assertTrue(result['ok'])
        self.assertEqual(result['tables'], len(manifest['tables']))

    def test_failed_capture_is_not_published(self):
        module = self.implementation()
        out = self.root / ('failed-' + uuid.uuid4().hex)
        with patch.object(module, 'dump_snapshot', side_effect=module.SnapshotError('dump-failed')):
            with self.assertRaises(module.SnapshotError):
                module.capture(self.container, self.database, 'math_master_preview', out, REVISION)
        self.assertFalse(out.exists())
        existing = self.root / ('existing-' + uuid.uuid4().hex)
        existing.mkdir(mode=0o700)
        marker = existing / 'retain'; marker.write_text('keep original backup')
        with self.assertRaises((ValueError, FileExistsError)):
            module.capture(self.container, self.database, 'math_master_preview', existing, REVISION)
        self.assertEqual(marker.read_text(), 'keep original backup')
        with patch.object(module, 'write_private', side_effect=OSError(errno.ENOSPC, 'simulated disk full')):
            with self.assertRaises(OSError):
                module.capture(self.container, self.database, 'math_master_preview', out, REVISION)
        self.assertFalse(out.exists())

    def test_rejects_corrupt_or_nonempty_target(self):
        module = self.implementation()
        out = self.backup('valid')
        with self.assertRaises(module.SnapshotError):
            module.restore(self.container, self.database, 'math_master_preview', out)
        with self.assertRaises(module.SnapshotError):
            module.restore(self.container, 'math_master_review_r1', 'math_master_preview', out)
        target, database = self.new_database()
        manifest = json.loads((out/'manifest.json').read_text())
        manifest['database']['collate'] = 'C'
        (out/'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, sort_keys=True)+'\n')
        hashes = [(hashlib.sha256((out/name).read_bytes()).hexdigest(), name) for name in ['database.dump','manifest.json']]
        (out/'SHA256SUMS').write_text(''.join(digest+'  '+name+'\n' for digest,name in hashes))
        with self.assertRaises(module.SnapshotError):
            module.restore(target, database, 'math_master_preview', out)
        self.assertEqual(module.sql(target, database, 'math_master_preview', "SELECT count(*) FROM pg_tables WHERE schemaname='public';").strip(), '0')
        with (out/'database.dump').open('ab') as stream:
            stream.write(b'corrupt')
        with self.assertRaises(module.SnapshotError):
            module.drill(out)

    def test_drill_cleans_objects_after_ambiguous_start_failure(self):
        module = self.implementation()
        out = self.backup('ambiguous-start')
        original = common.docker
        created = {}
        def transport(args, sudo=False, **kwargs):
            result = original(args, sudo, **kwargs)
            if args[0] == 'volume' and args[1] == 'create':
                created['volume'] = args[-1]
            if args[0] == 'run':
                created['container'] = args[args.index('--name')+1]
                result.returncode = 1  # daemon 已创建对象，但客户端报告启动失败。
            return result
        try:
            with patch.object(common, 'docker', side_effect=transport):
                with self.assertRaises(module.SnapshotError):
                    module.drill(out)
            self.assertEqual(original(['inspect',created['container']]).returncode, 1)
            self.assertEqual(original(['volume','inspect',created['volume']]).returncode, 1)
        finally:
            if 'container' in created: original(['rm','-f',created['container']])
            if 'volume' in created: original(['volume','rm',created['volume']])

    def test_drill_cleanup_keeps_foreign_volumes(self):
        module = self.implementation()
        out = self.backup('cleanup')
        foreign = 'math-master-foreign-' + uuid.uuid4().hex
        result = common.docker(['volume', 'create', '--label', 'math-master.restore-drill=foreign-owner', foreign])
        self.assertEqual(result.returncode, 0)
        try:
            result = module.drill(out)
            self.assertTrue(result['ok'])
            self.assertEqual(common.docker(['volume', 'inspect', foreign]).returncode, 0)
            self.assertEqual(common.docker(['inspect', result['container']]).returncode, 1)
            self.assertEqual(common.docker(['volume', 'inspect', result['volume']]).returncode, 1)
        finally:
            common.docker(['volume', 'rm', foreign])


class BackupCalendarTests(unittest.TestCase):
    def test_weekly_backup_uses_shanghai_calendar(self):
        self.assertIsNotNone(snapshot)
        function = getattr(snapshot, 'weekly_due', None)
        self.assertTrue(callable(function), 'Shanghai weekly backup calendar is missing')
        self.assertTrue(function(datetime.fromisoformat('2026-10-10T18:00:00+00:00')))
        self.assertFalse(function(datetime.fromisoformat('2026-10-11T18:00:00+00:00')))


if __name__ == '__main__':
    unittest.main()
