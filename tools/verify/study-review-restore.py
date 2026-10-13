"""仅用于随机隔离测试库的真实 14/15 快照、恢复和序列核对。"""
import importlib.util
from datetime import datetime
import json
import os
from pathlib import Path
import re
import sys
import tempfile
import time
import uuid

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'ops'))
import common
spec = importlib.util.spec_from_file_location('review_snapshot', ROOT / 'ops/database-snapshot.py')
snapshot = importlib.util.module_from_spec(spec)
spec.loader.exec_module(snapshot)


def sequences(container, database, user):
    names = snapshot.sql(container, database, user, "SELECT sequencename FROM pg_sequences WHERE schemaname='public' ORDER BY sequencename;").splitlines()
    return {name: json.loads(snapshot.sql(container, database, user, 'SELECT json_build_object(\'value\',last_value,\'called\',is_called) FROM public.' + snapshot.identifier(name))) for name in names}


def migration_rows(container, database, user):
    return json.loads(snapshot.sql(container, database, user, "SELECT coalesce(json_agg(row_to_json(t) ORDER BY id),'[]'::json) FROM public.goose_db_version t;"))


def verify_migration_increment(before, after, previous_sequence):
    # 旧记录的全部字段必须精确不变，仅允许 Goose 新增一次迁移 15 应用记录。
    if len(after) != len(before) + 1 or after[:-1] != before:
        raise ValueError('upgrade-original-migration-rows-changed')
    row = after[-1]
    flags = {'learning_enabled', 'correction_enabled', 'topic_study_enabled', 'topic_cutover_enabled', 'managed_knowledge_enabled'}
    if (set(row) != {'id', 'version_id', 'is_applied', 'tstamp'} | flags
            or type(row['id']) is not int or row['id'] != previous_sequence + 1
            or type(row['version_id']) is not int or row['version_id'] != 15
            or row['is_applied'] is not True or not isinstance(row['tstamp'], str)
            or any(row[flag] is not False for flag in flags)):
        raise ValueError('upgrade-migration-increment-invalid')
    try:
        datetime.fromisoformat(row['tstamp'])
    except ValueError as error:
        raise ValueError('upgrade-migration-timestamp-invalid') from error


def remove_target(target):
    container, token = target['container'], target['token']
    if not re.fullmatch(r'[0-9a-f]{32}', token) or container != 'math-master-review-restore-' + token:
        raise ValueError('unsafe-test-restore-target')
    check = common.docker(['inspect', '--format', '{{json .Config.Labels}}', container], False)
    if check.returncode == 0 and (json.loads(check.stdout) or {}).get(snapshot.LABEL) == token:
        if common.docker(['rm', '-f', container], False).returncode:
            raise ValueError('isolated-restore-cleanup-failed')


def restore_and_compare(backup, source_sequences, retain=None):
    data = snapshot.load_backup(backup)
    token = uuid.uuid4().hex
    container = 'math-master-review-restore-' + token
    database = 'math_master_restore_' + token
    user = 'math_master_preview'
    target = {'container': container, 'token': token, 'database': database, 'user': user}
    retained = False
    try:
        network = ['--publish', '127.0.0.1:0:5432'] if retain else ['--network', 'none']
        result = common.docker(['run', '--detach', '--platform', 'linux/amd64', *network, '--name', container,
                                '--label', snapshot.LABEL + '=' + token, '--tmpfs', '/var/lib/postgresql/data',
                                '-e', 'POSTGRES_USER=' + user, '-e', 'POSTGRES_DB=' + database,
                                '-e', 'POSTGRES_HOST_AUTH_METHOD=trust', '-e', 'POSTGRES_INITDB_ARGS=--encoding=UTF8 --lc-collate=' + data['database']['collate'] + ' --lc-ctype=' + data['database']['ctype'], snapshot.PG_IMAGE], False)
        if result.returncode:
            raise ValueError('isolated-restore-container-failed')
        for _ in range(150):
            if common.docker(['exec', container, 'pg_isready', '-h', '127.0.0.1', '-U', user, '-d', database], False).returncode == 0:
                break
            time.sleep(.2)
        else:
            raise ValueError('isolated-restore-readiness-failed')
        result = snapshot.restore(container, database, user, backup)
        if sequences(container, database, user) != source_sequences:
            raise ValueError('restored-sequences-not-identical')
        if retain:
            port = common.docker(['inspect', '--format', '{{(index (index .NetworkSettings.Ports "5432/tcp") 0).HostPort}}', container], False)
            if port.returncode or not re.fullmatch(rb'[0-9]+\n?', port.stdout):
                raise ValueError('missing-loopback-test-port')
            target['port'] = int(port.stdout)
            snapshot.write_private(retain, json.dumps(target).encode())
            retained = True
        return result
    finally:
        if not retained:
            remove_target(target)


def main():
    action, container, database, folder, revision = sys.argv[1:]
    if not re.fullmatch(r'math_master_test_[0-9a-f]{16}', database) or action not in ['capture14', 'upgrade15', 'capture15', 'cleanup14']:
        raise ValueError('unsafe-review-restore-source')
    snapshot.identity(container, database, 'postgres')
    folder = Path(folder)
    if action == 'cleanup14':
        remove_target(json.loads((folder / '14-target.json').read_text()))
        return
    if action.startswith('capture'):
        version = int(action[-2:])
        backup = folder / action
        snapshot.capture(container, database, 'postgres', backup, revision)
        data = snapshot.load_backup(backup)
        if data['migrationVersion'] != version:
            raise ValueError('wrong-source-migration')
        seq = sequences(container, database, 'postgres')
        snapshot.write_private(folder / (action + '-sequences.json'), json.dumps(seq).encode())
        if version == 14:
            snapshot.write_private(folder / 'capture14-migrations.json', json.dumps(migration_rows(container, database, 'postgres')).encode())
        result = restore_and_compare(backup, seq, folder / '14-target.json' if version == 14 else None)
        print(json.dumps({'ok': True, 'version': version, 'tables': result['tables'], 'sequences': len(seq)}))
        return
    target = json.loads((folder / '14-target.json').read_text())
    container, database, user = target['container'], target['database'], target['user']
    if snapshot.labels(container, False).get(snapshot.LABEL) != target['token']:
        raise ValueError('unowned-upgrade-target')
    previous = snapshot.load_backup(folder / 'capture14')
    actual = snapshot.inventory(container, database, user)
    if actual['migrationVersion'] != 15:
        raise ValueError('upgrade-not-15')
    new = {'study_review_sessions', 'study_review_snapshots', 'study_review_schedules', 'study_review_events', 'study_review_receipts'}
    if set(actual['tables']) - set(previous['tables']) != new or any(actual['tables'][name]['rows'] for name in new):
        raise ValueError('upgrade-new-tables-not-empty')
    for name, digest in previous['tables'].items():
        if name != 'goose_db_version' and actual['tables'][name] != digest:
            raise ValueError('upgrade-original-table-changed')
    seq = sequences(container, database, user)
    old_seq = json.loads((folder / 'capture14-sequences.json').read_text())
    verify_migration_increment(json.loads((folder / 'capture14-migrations.json').read_text()),
                               migration_rows(container, database, user), old_seq['goose_db_version_id_seq']['value'])
    if set(seq) != set(old_seq):
        raise ValueError('upgrade-original-sequence-inventory-changed')
    for name, value in old_seq.items():
        if name == 'goose_db_version_id_seq':
            if seq[name] != {'value': value['value'] + 1, 'called': True}:
                raise ValueError('upgrade-migration-sequence-changed')
        elif seq[name] != value:
            raise ValueError('upgrade-original-sequence-changed')
    print(json.dumps({'ok': True, 'version': 15, 'originalTables': len(previous['tables']), 'newEmptyTables': 5, 'sequences': len(seq)}))


if __name__ == '__main__':
    main()
