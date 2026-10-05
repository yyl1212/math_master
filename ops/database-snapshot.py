#!/usr/bin/env python3
"""一致性 PostgreSQL 快照、受保护恢复及私有备份轮换。"""
import argparse
from contextlib import contextmanager
import ctypes
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import time
import uuid
from zoneinfo import ZoneInfo

import common

PG_IMAGE = 'postgres:17.11@sha256:e31e3d5327d1806f6177827c9710643e4f35f7ab3f14d26d05332753d3e95ee0'
LABEL = 'math-master.restore-drill'
GUCS = "SET TIME ZONE 'UTC'; SET DateStyle='ISO,YMD'; SET bytea_output='hex'; SET extra_float_digits=3; SET statement_timeout='90s'; SET lock_timeout='5s';"


class SnapshotError(ValueError):
    pass


def identifier(value: str) -> str:
    return '"' + value.replace('"', '""') + '"'


def identity(container: str, database: str, user: str) -> None:
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]{0,127}', container) or any(not re.fullmatch(r'[a-z][a-z0-9_]{0,62}', v) for v in [database, user]):
        raise SnapshotError('invalid-database-identity')


def psql_args(container: str, database: str, user: str) -> list[str]:
    identity(container, database, user)
    return ['exec', '-i', '-e', 'PGCLIENTENCODING=UTF8', container, 'psql', '-X', '-qAt', '-v', 'ON_ERROR_STOP=1', '-U', user, '-d', database]


def sql(container: str, database: str, user: str, query: str, sudo: bool = False, *, snapshot: str | None = None) -> str:
    prefix = GUCS
    if snapshot is not None:
        if not re.fullmatch(r'[0-9A-Fa-f]{8}-[0-9A-Fa-f]{8}-[0-9]+', snapshot):
            raise SnapshotError('invalid-snapshot-id')
        prefix = "BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY; SET TRANSACTION SNAPSHOT '" + snapshot + "';" + GUCS
    result = common.docker([*psql_args(container, database, user), '-c', prefix + query], sudo)
    if result.returncode:
        raise SnapshotError('database-query-failed')
    return result.stdout.decode('utf-8').strip()


@contextmanager
def exported_snapshot(container: str, database: str, user: str, sudo: bool):
    command = common.docker_command(psql_args(container, database, user), sudo)
    with tempfile.TemporaryFile() as errors:
        keeper = subprocess.Popen(command, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=errors, env=common.clean_environment())
        try:
            keeper.stdin.write(("SET idle_in_transaction_session_timeout='500s'; BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY; SELECT pg_export_snapshot();\n").encode())
            keeper.stdin.flush()
            token = keeper.stdout.readline().decode().strip()
            if not re.fullmatch(r'[0-9A-Fa-f]{8}-[0-9A-Fa-f]{8}-[0-9]+', token):
                raise SnapshotError('snapshot-export-failed')
            yield token
        finally:
            try:
                keeper.stdin.write(b'ROLLBACK;\n'); keeper.stdin.flush(); keeper.stdin.close()
                keeper.wait(timeout=5)
            except (OSError, subprocess.TimeoutExpired):
                keeper.kill(); keeper.wait(timeout=5)
            keeper.stdout.close()


def database_metadata(container: str, database: str, user: str, sudo: bool, snapshot: str | None = None) -> dict:
    query = "SELECT json_build_object('name',datname,'encoding',pg_encoding_to_char(encoding),'collate',datcollate,'ctype',datctype,'localeProvider',datlocprovider,'versionNum',current_setting('server_version_num')::int) FROM pg_database WHERE datname=current_database();"
    result = json.loads(sql(container, database, user, query, sudo, snapshot=snapshot))
    if result['name'] != database or result['encoding'] != 'UTF8' or result['localeProvider'] != 'c' or result['versionNum'] != 170011:
        raise SnapshotError('unsupported-database-metadata')
    result.pop('name')
    return result


def table_digest(container: str, database: str, user: str, table: str, sudo: bool, snapshot: str | None) -> dict:
    prefix = GUCS
    if snapshot:
        prefix = "BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY; SET TRANSACTION SNAPSHOT '" + snapshot + "';" + GUCS
    query = 'SELECT row_to_json(t)::text AS row FROM public.' + identifier(table) + ' t ORDER BY row_to_json(t)::text COLLATE "C";'
    command = common.docker_command([*psql_args(container, database, user), '-c', prefix + query], sudo)
    digest = hashlib.sha256(); rows = 0
    with tempfile.TemporaryFile() as errors:
        process = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=errors, env=common.clean_environment())
        try:
            for block in iter(lambda: process.stdout.read(65536), b''):
                digest.update(block); rows += block.count(b'\n')
            if process.wait(timeout=5):
                raise SnapshotError('table-inventory-failed')
        finally:
            process.stdout.close()
            if process.poll() is None:
                process.kill(); process.wait(timeout=5)
    return {'rows': rows, 'sha256': digest.hexdigest()}


def inventory(container: str, database: str, user: str, sudo: bool = False, snapshot: str | None = None) -> dict:
    metadata = database_metadata(container, database, user, sudo, snapshot)
    names = sql(container, database, user, "SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename COLLATE \"C\";", sudo, snapshot=snapshot).splitlines()
    tables = {name: table_digest(container, database, user, name, sudo, snapshot) for name in names}
    migration = int(sql(container, database, user, 'SELECT coalesce(max(version_id) FILTER (WHERE is_applied),0) FROM public.goose_db_version;', sudo, snapshot=snapshot)) if 'goose_db_version' in tables else 0
    workspaces = []
    for table, kind in [('content_workspaces', 'knowledge'), ('question_workspaces', 'questions')]:
        if table in tables:
            query = "SELECT coalesce(json_agg(json_build_object('kind','" + kind + "','id',id::text,'revision',revision,'status',status,'packageId',package->>'id','packageVersion',package->'version') ORDER BY id),'[]'::json) FROM public." + identifier(table) + ';'
            workspaces.extend(json.loads(sql(container, database, user, query, sudo, snapshot=snapshot)))
    roles = []
    if 'auth_users' in tables and 'auth_user_roles' in tables:
        query = "SELECT coalesce(json_agg(v ORDER BY id),'[]'::json) FROM (SELECT u.id::text id,u.username,coalesce(array_agg(r.role ORDER BY r.role) FILTER (WHERE r.role IS NOT NULL),ARRAY[]::text[]) roles FROM auth_users u LEFT JOIN auth_user_roles r ON r.user_id=u.id GROUP BY u.id,u.username) v;"
        roles = json.loads(sql(container, database, user, query, sudo, snapshot=snapshot))
    assets = []
    if 'content_workspace_assets' in tables:
        query = "SELECT coalesce(json_agg(json_build_object('workspaceId',workspace_id::text,'assetId',asset_id,'sha256',sha256,'byteSha256',encode(sha256(bytes),'hex')) ORDER BY workspace_id,asset_id),'[]'::json) FROM content_workspace_assets;"
        assets = json.loads(sql(container, database, user, query, sudo, snapshot=snapshot))
    heads = {name: tables[name]['rows'] if name in tables else 0 for name in ['publication_heads', 'question_heads']}
    return {'database': metadata, 'tables': tables, 'migrationVersion': migration, 'workspaces': workspaces, 'roles': roles, 'assets': assets, 'heads': heads}


def write_private(path: Path, data: bytes) -> None:
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'wb') as stream:
        stream.write(data); stream.flush(); os.fsync(stream.fileno())


def digest_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(65536), b''):
            digest.update(block)
    return digest.hexdigest()


def publish_directory(stage: Path, out: Path) -> None:
    # 两个实际支持的平台均使用内核的“目标不存在才改名”，不会覆盖并发创建的目录。
    libc = ctypes.CDLL(None, use_errno=True)
    source, destination = os.fsencode(stage), os.fsencode(out)
    if sys.platform == 'linux':
        operation = libc.renameat2
        operation.argtypes = [ctypes.c_int, ctypes.c_char_p, ctypes.c_int, ctypes.c_char_p, ctypes.c_uint]
        result = operation(-100, source, -100, destination, 1)
    elif sys.platform == 'darwin':
        operation = libc.renamex_np
        operation.argtypes = [ctypes.c_char_p, ctypes.c_char_p, ctypes.c_uint]
        result = operation(source, destination, 4)
    else:
        raise SnapshotError('unsupported-atomic-publication')
    if result:
        raise OSError(ctypes.get_errno(), 'atomic-publication-failed')


def dump_snapshot(container: str, database: str, user: str, token: str, out: Path, sudo: bool) -> None:
    command = common.docker_command(['exec', '-e', 'PGCLIENTENCODING=UTF8', container, 'pg_dump', '-U', user, '-d', database, '--format=custom', '--encoding=UTF8', '--snapshot=' + token], sudo)
    fd = os.open(out, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'wb') as output, tempfile.TemporaryFile() as errors:
        result = subprocess.run(command, stdout=output, stderr=errors, env=common.clean_environment(), timeout=530)
        output.flush(); os.fsync(output.fileno())
    if result.returncode:
        raise SnapshotError('database-dump-failed')


def capture(container: str, database: str, user: str, out: Path, revision: str, sudo: bool = False, *, kind: str = 'manual') -> dict:
    identity(container, database, user); common.validate_revision(revision)
    out = common.safe_path(out)
    if out.exists():
        raise FileExistsError('backup-already-exists')
    out.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
    stage = Path(tempfile.mkdtemp(prefix='.capture-', dir=out.parent))
    try:
        with exported_snapshot(container, database, user, sudo) as token:
            dump_snapshot(container, database, user, token, stage/'database.dump', sudo)
            data = inventory(container, database, user, sudo, token)
        data.update(schemaVersion=1, sourceCommit=revision, createdAt=datetime.now(timezone.utc).isoformat(), backupKind=kind, dumpSha256=digest_file(stage/'database.dump'))
        write_private(stage/'manifest.json', (json.dumps(data, ensure_ascii=False, sort_keys=True, indent=2)+'\n').encode())
        sums = ''.join(digest_file(stage/name)+'  '+name+'\n' for name in ['database.dump','manifest.json'])
        write_private(stage/'SHA256SUMS', sums.encode())
        publish_directory(stage, out)
        return {'ok': True, 'tables': len(data['tables']), 'workspaces': len(data['workspaces']), 'dumpSha256': data['dumpSha256']}
    finally:
        if stage.exists():
            shutil.rmtree(stage)


def load_backup(backup: Path) -> dict:
    backup = common.safe_path(backup)
    if stat.S_IMODE(backup.stat().st_mode) != 0o700 or backup.stat().st_uid != os.geteuid():
        raise SnapshotError('unsafe-backup-directory')
    for name in ['database.dump','manifest.json','SHA256SUMS']:
        path = common.safe_path(backup/name); metadata = path.lstat()
        if not stat.S_ISREG(metadata.st_mode) or stat.S_IMODE(metadata.st_mode) != 0o600 or metadata.st_uid != os.geteuid():
            raise SnapshotError('unsafe-backup-file')
    expected = (backup/'SHA256SUMS').read_text().splitlines()
    actual = [digest_file(backup/name)+'  '+name for name in ['database.dump','manifest.json']]
    if expected != actual:
        raise SnapshotError('backup-checksum-mismatch')
    with (backup/'database.dump').open('rb') as stream:
        if stream.read(5) != b'PGDMP':
            raise SnapshotError('invalid-dump-format')
    data = json.loads((backup/'manifest.json').read_text())
    if data.get('schemaVersion') != 1 or data.get('dumpSha256') != actual[0].split()[0] or not isinstance(data.get('tables'),dict):
        raise SnapshotError('invalid-backup-manifest')
    common.validate_revision(data['sourceCommit'])
    return data


def verify_sequences(container: str, database: str, user: str, sudo: bool) -> None:
    query = "SELECT coalesce(json_agg(json_build_object('sequence',s.relname,'table',t.relname,'column',a.attname,'increment',q.seqincrement)),'[]'::json) FROM pg_class s JOIN pg_namespace sn ON sn.oid=s.relnamespace JOIN pg_depend d ON d.objid=s.oid AND d.deptype IN ('a','i') JOIN pg_class t ON t.oid=d.refobjid JOIN pg_namespace tn ON tn.oid=t.relnamespace JOIN pg_attribute a ON a.attrelid=t.oid AND a.attnum=d.refobjsubid JOIN pg_sequence q ON q.seqrelid=s.oid WHERE s.relkind='S' AND sn.nspname='public' AND tn.nspname='public';"
    for sequence in json.loads(sql(container,database,user,query,sudo)):
        values = json.loads(sql(container,database,user,'SELECT json_build_object(\'value\',last_value,\'called\',is_called) FROM public.'+identifier(sequence['sequence'])+';',sudo))
        maximum = int(sql(container,database,user,'SELECT coalesce(max('+identifier(sequence['column'])+'),0) FROM public.'+identifier(sequence['table'])+';',sudo))
        increment = sequence['increment']
        if increment <= 0 or values['value'] + (increment if values['called'] else 0) <= maximum:
            raise SnapshotError('restored-sequence-behind-data')


def inspect(container: str, database: str, user: str, backup: Path, sudo: bool = False) -> dict:
    data = load_backup(backup)
    actual = inventory(container,database,user,sudo)
    for key in ['database','tables','migrationVersion','workspaces','roles','assets','heads']:
        if actual[key] != data[key]:
            raise SnapshotError('restored-inventory-mismatch')
    verify_sequences(container,database,user,sudo)
    return {'ok': True, 'tables': len(data['tables']), 'workspaces': len(data['workspaces']), 'assets': len(data['assets'])}


def labels(container: str, sudo: bool) -> dict:
    result = common.docker(['inspect','--format','{{json .Config.Labels}}',container],sudo)
    if result.returncode:
        raise SnapshotError('unknown-restore-target')
    return json.loads(result.stdout) or {}


def restore(container: str, database: str, user: str, backup: Path, sudo: bool = False) -> dict:
    identity(container,database,user)
    data = load_backup(backup)
    tags = labels(container,sudo)
    production = tags.get('com.docker.compose.project') == common.PROJECT and tags.get('com.docker.compose.service') == 'db' and database == 'math_master_preview'
    token = tags.get(LABEL,'')
    isolated = bool(re.fullmatch(r'[0-9a-f]{32}',token)) and database == 'math_master_restore_' + token
    if not (production or isolated) or user != 'math_master_preview':
        raise SnapshotError('unsafe-restore-target')
    if database_metadata(container,database,user,sudo) != data['database']:
        raise SnapshotError('database-metadata-mismatch')
    empty = sql(container,database,user,"SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind IN ('r','p','v','m','S','f');",sudo)
    if empty != '0':
        raise SnapshotError('nonempty-restore-target')
    command = common.docker_command(['exec','-i',container,'pg_restore','-U',user,'-d',database,'--no-owner','--no-acl','--exit-on-error','--single-transaction'],sudo)
    with (backup/'database.dump').open('rb') as source, tempfile.TemporaryFile() as errors:
        result = subprocess.run(command,stdin=source,stdout=subprocess.DEVNULL,stderr=errors,env=common.clean_environment(),timeout=530)
    if result.returncode:
        raise SnapshotError('database-restore-failed')
    return inspect(container,database,user,backup,sudo)


def drill(backup: Path, sudo: bool = False) -> dict:
    data = load_backup(backup)
    token = uuid.uuid4().hex
    container = 'math-master-restore-' + token
    volume = 'math-master-restore-' + token
    database = 'math_master_restore_' + token
    created_container = False; created_volume = False
    with tempfile.TemporaryDirectory() as directory:
        env = Path(directory)/'postgres.env'
        settings = 'POSTGRES_USER=math_master_preview\nPOSTGRES_PASSWORD='+os.urandom(32).hex()+'\nPOSTGRES_DB='+database+'\nPOSTGRES_INITDB_ARGS=--encoding=UTF8 --lc-collate='+data['database']['collate']+' --lc-ctype='+data['database']['ctype']+'\n'
        write_private(env,settings.encode())
        try:
            created_volume = True
            result = common.docker(['volume','create','--label',LABEL+'='+token,volume],sudo)
            if result.returncode:
                raise SnapshotError('drill-volume-failed')
            created_volume = True
            created_container = True
            result = common.docker(['run','--detach','--platform','linux/amd64','--network','none','--name',container,'--label',LABEL+'='+token,'--env-file',str(env),'--mount','type=volume,src='+volume+',dst=/var/lib/postgresql/data',PG_IMAGE],sudo)
            if result.returncode:
                raise SnapshotError('drill-container-failed')
            created_container = True
            for _ in range(60):
                if common.docker(['exec',container,'pg_isready','-h','127.0.0.1','-U','math_master_preview','-d',database],sudo).returncode == 0:
                    break
                time.sleep(.2)
            else:
                raise SnapshotError('drill-readiness-failed')
            result = restore(container,database,'math_master_preview',backup,sudo)
            return {**result,'container':container,'volume':volume}
        finally:
            if created_container:
                owned = common.docker(['inspect','--format','{{json .Config.Labels}}',container],sudo)
                if owned.returncode == 0 and (json.loads(owned.stdout) or {}).get(LABEL) == token:
                    if common.docker(['rm','-f',container],sudo).returncode:
                        raise SnapshotError('drill-cleanup-failed')
            if created_volume:
                result = common.docker(['volume','inspect','--format','{{json .Labels}}',volume],sudo)
                if result.returncode == 0 and (json.loads(result.stdout) or {}).get(LABEL) == token:
                    if common.docker(['volume','rm',volume],sudo).returncode:
                        raise SnapshotError('drill-cleanup-failed')


def prune(folder: Path, kind: str, keep: int) -> None:
    groups: dict[str,list[tuple[str,Path]]] = {}
    for path in folder.iterdir():
        if not re.fullmatch(r'[0-9]{8}T[0-9]{6}Z-[0-9a-f]{8}',path.name) or not path.is_dir() or path.is_symlink():
            continue
        try:
            data = load_backup(path)
            if data.get('backupKind') != kind:
                continue
            stamp = datetime.fromisoformat(data['createdAt'])
            group = stamp.strftime('%Y-%m-%d') if kind == 'daily' else '%04d-%02d'%stamp.isocalendar()[:2]
            groups.setdefault(group,[]).append((data['createdAt'],path))
        except (ValueError,OSError,KeyError):
            continue
    retained = set(sorted(groups)[-keep:])
    for group, entries in groups.items():
        latest = max(entries)[1] if group in retained else None
        for _, path in entries:
            if path != latest:
                shutil.rmtree(path)


def weekly_due(stamp: datetime) -> bool:
    return stamp.astimezone(ZoneInfo('Asia/Shanghai')).weekday() == 6


def backup(root: Path, kind: str, sudo: bool = False, revision: str | None = None) -> dict:
    root = common.safe_path(root)
    if kind not in ['daily','weekly','predeploy']:
        raise SnapshotError('invalid-backup-kind')
    if revision is None:
        state = json.loads(common.safe_path(root/'shared'/'deployment.json').read_text())
        revision = state.get('runningRevision') or state.get('current')
    common.validate_revision(revision or '')
    resolved = common.compose(root,revision,['ps','-q','db'],sudo)
    if resolved.returncode or not re.fullmatch(rb'[0-9a-f]{64}',resolved.stdout.strip()):
        raise SnapshotError('preview-database-unavailable')
    container = resolved.stdout.decode().strip()
    out = root/'backups'/kind/(datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ')+'-'+uuid.uuid4().hex[:8])
    result = capture(container,'math_master_preview','math_master_preview',out,revision,sudo,kind=kind)
    if kind in ['daily','weekly']:
        prune(out.parent,kind,7 if kind=='daily' else 4)
    if kind=='daily' and weekly_due(datetime.now(timezone.utc)):
        backup(root,'weekly',sudo,revision)
    return {**result,'backup':str(out)}


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='operation',required=True)
    for operation in ['capture','inspect','restore']:
        item = sub.add_parser(operation)
        for flag in ['container','database','user']:
            item.add_argument('--'+flag,required=True)
        item.add_argument('--sudo',action='store_true')
        if operation=='capture':
            item.add_argument('--out',type=Path,required=True); item.add_argument('--revision',required=True)
        else:
            item.add_argument('--backup',type=Path,required=True)
    item=sub.add_parser('drill');item.add_argument('--backup',type=Path,required=True);item.add_argument('--sudo',action='store_true')
    item=sub.add_parser('backup');item.add_argument('--root',type=Path,required=True);item.add_argument('--kind',choices=['daily','weekly','predeploy'],required=True);item.add_argument('--revision');item.add_argument('--sudo',action='store_true')
    args=parser.parse_args(argv)
    try:
        if args.operation=='capture': result=capture(args.container,args.database,args.user,args.out,args.revision,args.sudo)
        elif args.operation in ['inspect','restore']: result=globals()[args.operation](args.container,args.database,args.user,args.backup,args.sudo)
        elif args.operation=='drill': result=drill(args.backup,args.sudo)
        else: result=backup(args.root,args.kind,args.sudo,args.revision)
        print(json.dumps(result));return 0
    except (ValueError,OSError,KeyError,subprocess.SubprocessError) as error:
        stage=str(error) if re.fullmatch(r'[a-z-]{1,64}',str(error)) else 'snapshot-operation-failed'
        print(json.dumps({'ok':False,'stage':stage}),file=sys.stderr);return 1


if __name__=='__main__':
    sys.exit(main())
