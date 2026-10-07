#!/usr/bin/env python3
"""固定提交的服务器部署状态机；失败时保留数据库与诊断状态。"""
import argparse
from contextlib import contextmanager
from datetime import datetime, timezone
import fcntl
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import uuid

import common

spec=importlib.util.spec_from_file_location('database_snapshot',Path(__file__).with_name('database-snapshot.py'))
snapshot=importlib.util.module_from_spec(spec);spec.loader.exec_module(snapshot)


class DeployError(ValueError):
    pass


def topicSchemaCompatibility(actualMigration: int,targetCapabilities: dict,mode: str) -> bool:
    if type(actualMigration) is not int or not 1<=actualMigration<=12 or mode not in ['legacy','topics']:return False
    if set(targetCapabilities)!={'taxonomy','study','retirement'} or any(type(v) is not bool for v in targetCapabilities.values()):return False
    if mode=='topics' and actualMigration<12:return False
    required={'taxonomy':actualMigration>=10,'study':actualMigration>=11,'retirement':actualMigration>=12 or mode=='topics'}
    return all(not needed or targetCapabilities[name] for name,needed in required.items())


def topic_database_state(container: str,sudo: bool) -> dict:
    version=int(snapshot.sql(container,'math_master_preview','math_master_preview','SELECT coalesce(max(version_id) FILTER (WHERE is_applied),0) FROM goose_db_version;',sudo))
    names=json.loads(snapshot.sql(container,'math_master_preview','math_master_preview',"SELECT coalesce(json_agg(attname),'[]'::json) FROM pg_attribute WHERE attrelid=to_regclass('public.goose_db_version') AND attname IN ('topic_study_enabled','topic_cutover_enabled') AND NOT attisdropped;",sudo))
    required=version
    for column,minimum in [('topic_study_enabled',11),('topic_cutover_enabled',12)]:
        if column in names:
            value=snapshot.sql(container,'math_master_preview','math_master_preview','SELECT coalesce(bool_or('+column+'),false) FROM goose_db_version WHERE version_id=0;',sudo).strip()
            if value=='t':required=max(required,minimum)
            elif value!='f':raise DeployError('topic-state-unavailable')
    exists=snapshot.sql(container,'math_master_preview','math_master_preview',"SELECT to_regclass('public.topic_learning_state') IS NOT NULL;",sudo).strip()
    if exists=='t':mode=snapshot.sql(container,'math_master_preview','math_master_preview','SELECT experience_mode FROM topic_learning_state WHERE singleton;',sudo).strip()
    elif exists=='f' and required<10:mode='legacy'
    else:raise DeployError('topic-state-unavailable')
    if mode not in ['legacy','topics']:raise DeployError('topic-state-unavailable')
    return {'requiredMigration':required,'mode':mode}


def runtime_topic_health(root: Path,revision: str,sudo: bool) -> dict:
    raw=run_compose(root,revision,['exec','-T','api','wget','-qO-','http://127.0.0.1:8080/readyz'],sudo,'topic-readiness-unavailable')
    if len(raw)>8192:raise DeployError('topic-readiness-unavailable')
    result=json.loads(raw);topic=result.get('topic')
    if topic is None:return {'taxonomy':False,'study':False,'retirement':False,'schemaReady':result.get('status')=='ready','topicsMode':False}
    if set(topic)!={'taxonomy','study','retirement','schemaReady','topicsMode'} or any(type(v) is not bool for v in topic.values()) or result.get('status')!='ready':raise DeployError('topic-readiness-unavailable')
    return topic


def verify_topic_runtime(root: Path,revision: str,sudo: bool,container: str) -> None:
    actual=topic_database_state(container,sudo);health=runtime_topic_health(root,revision,sudo)
    capabilities={key:health[key] for key in ['taxonomy','study','retirement']}
    if not health['schemaReady'] or health['topicsMode']!=(actual['mode']=='topics') or not topicSchemaCompatibility(actual['requiredMigration'],capabilities,actual['mode']):raise DeployError('topic-schema-incompatible')


def load_state(root: Path) -> dict:
    path=common.safe_path(root/'shared'/'deployment.json')
    if not path.exists():
        return {'schemaVersion':1,'current':None,'previous':None,'runningRevision':None,'bootstrapVerified':False,'status':'uninitialized'}
    info=path.lstat()
    if not stat.S_ISREG(info.st_mode) or stat.S_IMODE(info.st_mode)!=0o600 or info.st_uid!=os.geteuid():
        raise DeployError('unsafe-deployment-state')
    value=json.loads(path.read_text())
    if value.get('schemaVersion')!=1:
        raise DeployError('invalid-deployment-state')
    for field in ['current','previous','runningRevision']:
        if value.get(field) is not None:common.validate_revision(value[field])
    return value


def save_state(root: Path,state: dict) -> None:
    target=common.safe_path(root/'shared'/'deployment.json')
    temporary=target.with_name('.deployment-'+uuid.uuid4().hex)
    state['updatedAt']=datetime.now(timezone.utc).isoformat()
    snapshot.write_private(temporary,(json.dumps(state,sort_keys=True,indent=2)+'\n').encode())
    os.replace(temporary,target)


@contextmanager
def deployment_lock(root: Path):
    path=common.safe_path(root/'shared'/'deployment.lock')
    descriptor=os.open(path,os.O_RDWR|os.O_CREAT|os.O_NOFOLLOW,0o600)
    with os.fdopen(descriptor,'r+') as stream:
        try:fcntl.flock(stream,fcntl.LOCK_EX|fcntl.LOCK_NB)
        except BlockingIOError:raise DeployError('deployment-already-running') from None
        yield


def run_compose(root: Path,revision: str,args: list[str],sudo: bool,stage: str,acme: str='production') -> bytes:
    result=common.compose(root,revision,args,sudo,acme)
    if result.returncode:raise DeployError(stage)
    return result.stdout


def database_container(root: Path,revision: str,sudo: bool) -> str:
    value=run_compose(root,revision,['ps','-q','db'],sudo,'database-unavailable').decode().strip()
    if not re.fullmatch(r'[0-9a-f]{64}',value):raise DeployError('database-unavailable')
    return value


def expected_migration(root: Path,revision: str) -> int:
    folder=common.safe_path(root/'releases'/revision/'db'/'migrations')
    versions=[int(p.name.split('_',1)[0]) for p in folder.iterdir() if re.fullmatch(r'[0-9]{5}_[a-z0-9_]+\.sql',p.name) and not p.is_symlink()]
    if not versions:raise DeployError('missing-migrations')
    return max(versions)


def trusted_check(origin: str) -> dict:
    path=Path(__file__).with_name('verify-deployment.py')
    if not path.is_file():raise DeployError('deployment-verifier-unavailable')
    spec=importlib.util.spec_from_file_location('deployment_verifier',path)
    verifier=importlib.util.module_from_spec(spec);spec.loader.exec_module(verifier)
    result=verifier.check(origin,1,1)
    if not result.get('ok'):raise DeployError('trusted-https-check-failed')
    return result


def publish_current(root: Path,revision: str,state: dict) -> None:
    pointer=root/'current'
    if pointer.exists() or pointer.is_symlink():
        if not pointer.is_symlink() or not re.fullmatch(r'releases/[0-9a-f]{40}',pointer.readlink().as_posix()):
            raise DeployError('unsafe-current-pointer')
    temporary=root/('.current-'+uuid.uuid4().hex)
    temporary.symlink_to('releases/'+revision)
    os.replace(temporary,pointer)
    previous=state.get('current')
    state.update(current=revision,previous=previous if previous!=revision else state.get('previous'),runningRevision=revision,status='active')
    save_state(root,state)


def stop_writes(root: Path,revision: str,sudo: bool) -> None:
    run_compose(root,revision,['stop','gateway','web','api'],sudo,'stop-writes-failed')


def rollback(root: Path,revision: str,sudo: bool,state: dict) -> None:
    if revision not in [state.get('current'),state.get('previous')]:raise DeployError('unknown-rollback-release')
    try:
        stop_writes(root,state.get("runningRevision") or revision,sudo)
        container=database_container(root,revision,sudo)
        migration=int(snapshot.sql(container,'math_master_preview','math_master_preview','SELECT coalesce(max(version_id) FILTER (WHERE is_applied),0) FROM goose_db_version;',sudo))
        if migration!=expected_migration(root,revision):raise DeployError('schema-incompatible')
        run_compose(root,revision,['up','-d','--no-deps','--wait','--wait-timeout','120','api'],sudo,'rollback-health-failed')
        verify_topic_runtime(root,revision,sudo,container)
        run_compose(root,revision,['up','-d','--no-deps','--wait','--wait-timeout','120','api','web','gateway'],sudo,'rollback-health-failed')
        trusted_check(common.ORIGIN)
        publish_current(root,revision,state)
    except (ValueError,OSError,subprocess.SubprocessError) as error:
        try:
            stop_writes(root,revision,sudo)
        finally:
            state.update(status='manual-recovery-required',failureStage=str(error) if isinstance(error,DeployError) else 'rollback-failed')
            save_state(root,state)
        raise


def recover(root: Path,revision: str,sudo: bool,state: dict) -> None:
    previous=state.get('current')
    try:
        if previous:
            rollback(root,previous,sudo,state)
        else:
            stop_writes(root,revision,sudo)
            state.update(status='stopped-after-failure',runningRevision=None)
            save_state(root,state)
    except (ValueError,OSError,subprocess.SubprocessError):
        stop_writes(root,revision,sudo)
        state.update(status='manual-recovery-required')
        save_state(root,state)


def prepare(root: Path,revision: str,sudo: bool,state: dict) -> None:
    source=common.safe_path(root/'shared'/'.env')
    common.load_env(source)
    configurations=root/'shared'/'configs';configurations.mkdir(mode=0o700,exist_ok=True)
    target=common.safe_path(configurations/(revision+'.env'))
    original=source.read_bytes()
    if target.exists():
        common.load_env(target)
        if target.read_bytes()!=original:raise DeployError('release-config-already-frozen')
    else:snapshot.write_private(target,original)
    state.update(attempt=revision,attemptConfigSha256=hashlib.sha256(original).hexdigest(),status='preparing')
    save_state(root,state)
    for service in ['api','web','gateway']:
        run_compose(root,revision,['build',service],sudo,'build-'+service+'-failed')
    run_compose(root,revision,['up','-d','--wait','--wait-timeout','120','db'],sudo,'database-health-failed')
    state.update(status='prepared');save_state(root,state)


def start(root: Path,revision: str,sudo: bool,state: dict,baseline: Path | None) -> None:
    container=database_container(root,revision,sudo)
    source_revision=state.get('runningRevision') or state.get('current') or state.get('baselineSourceCommit')
    if not state.get('bootstrapVerified'):
        if baseline is None:raise DeployError('first-start-requires-baseline')
        data=snapshot.load_backup(baseline)
        if data['migrationVersion']!=expected_migration(root,revision):raise DeployError('baseline-schema-mismatch')
        snapshot.inspect(container,'math_master_preview','math_master_preview',baseline,sudo)
        state.update(bootstrapVerified=True,baselineDumpSha256=data['dumpSha256'],baselineSourceCommit=data['sourceCommit'])
        source_revision=data['sourceCommit'];save_state(root,state)
    if source_revision is None:raise DeployError('unknown-database-provenance')
    backup=root/'backups'/'predeploy'/(datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ')+'-'+uuid.uuid4().hex[:8])
    try:
        stop_writes(root,revision,sudo)
        snapshot.capture(container,'math_master_preview','math_master_preview',backup,source_revision,sudo,kind='predeploy')
        run_compose(root,revision,['run','--rm','--no-deps','--entrypoint','/app/bin/migrate','api','--dir','/app/db/migrations','up'],sudo,'migration-failed')
        run_compose(root,revision,['up','-d','--no-deps','--wait','--wait-timeout','120','api'],sudo,'application-health-failed')
        verify_topic_runtime(root,revision,sudo,container)
        run_compose(root,revision,['up','-d','--no-deps','--wait','--wait-timeout','120','api','web'],sudo,'application-health-failed')
        state.update(attempt=revision,runningRevision=revision,status='ready');save_state(root,state)
    except (ValueError,OSError,subprocess.SubprocessError):
        recover(root,revision,sudo,state)
        raise


def activate(root: Path,revision: str,sudo: bool,state: dict,acme: str) -> None:
    if not state.get('bootstrapVerified') or state.get('runningRevision')!=revision:
        raise DeployError('application-not-ready-for-activation')
    try:
        verify_topic_runtime(root,revision,sudo,database_container(root,revision,sudo))
        # 强制重建，确保切换 issuer 和相应证书卷，不延用 staging 容器。
        run_compose(root,revision,['up','-d','--no-deps','--force-recreate','--wait','--wait-timeout','120','gateway'],sudo,'gateway-health-failed',acme)
        if acme=='production':
            trusted_check(common.ORIGIN);publish_current(root,revision,state)
        else:
            state.update(status='staging-started');save_state(root,state)
    except (ValueError,OSError,subprocess.SubprocessError):
        recover(root,revision,sudo,state)
        raise


def main(argv: list[str] | None=None) -> int:
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('operation',choices=['prepare','start','activate','rollback'])
    parser.add_argument('--root',required=True,type=Path);parser.add_argument('--revision',required=True)
    parser.add_argument('--sudo',action='store_true');parser.add_argument('--baseline',type=Path)
    parser.add_argument('--acme',choices=['staging','production'])
    args=parser.parse_args(argv)
    try:
        revision=common.validate_revision(args.revision)
        root=common.safe_path(args.root)
        release=common.safe_path(root/'releases'/revision)
        if not (release/'compose.yaml').is_file() or not (root/'shared').is_dir():raise DeployError('missing-release')
        if args.operation=='activate' and args.acme is None:raise DeployError('acme-selection-required')
        with deployment_lock(root):
            state=load_state(root)
            if args.operation=='prepare':prepare(root,revision,args.sudo,state)
            elif args.operation=='start':start(root,revision,args.sudo,state,args.baseline)
            elif args.operation=='activate':activate(root,revision,args.sudo,state,args.acme)
            else:rollback(root,revision,args.sudo,state)
        print(json.dumps({'ok':True,'stage':args.operation,'revision':revision}));return 0
    except (ValueError,OSError,KeyError,subprocess.SubprocessError) as error:
        stage=str(error) if re.fullmatch(r'[a-z-]{1,64}',str(error)) else 'deployment-operation-failed'
        print(json.dumps({'ok':False,'stage':stage}),file=sys.stderr);return 1


if __name__=='__main__':sys.exit(main())
