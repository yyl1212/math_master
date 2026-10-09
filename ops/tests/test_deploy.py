"""真实私有目录与外部 Docker 边界模拟，验证部署状态及失败回退。"""
from contextlib import redirect_stdout, redirect_stderr
import fcntl
import importlib.util
import io
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

OPS=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(OPS))
import common
from test_config import BODY
module=None
if (OPS/'deploy.py').exists():
    spec=importlib.util.spec_from_file_location('deploy',OPS/'deploy.py')
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
OLD='3'*40
NEW='4'*40
SCHEMA=max(int(p.name.split('_',1)[0]) for p in (OPS.parent/'db'/'migrations').glob('*.sql'))


class DeployTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory(dir='/private/tmp' if Path('/private/tmp').exists() else None)
        self.addCleanup(self.temp.cleanup)
        self.root=Path(self.temp.name)
        shared=self.root/'shared';shared.mkdir(mode=0o700)
        (shared/'configs').mkdir(mode=0o700)
        (shared/'.env').write_text(BODY);(shared/'.env').chmod(0o600)
        for revision in [OLD,NEW]:
            release=self.root/'releases'/revision;release.mkdir(parents=True)
            shutil.copyfile(OPS.parent/'compose.yaml',release/'compose.yaml')
            shutil.copytree(OPS.parent/'db'/'migrations',release/'db'/'migrations')
            config=shared/'configs'/(revision+'.env');config.write_text(BODY);config.chmod(0o600)
        profile=patch.object(module,'topic_database_state',return_value={'requiredMigration':SCHEMA,'mode':'legacy'})
        profile.start();self.addCleanup(profile.stop)
        self.commands=[]
        self.state_path=shared/'deployment.json'
        self.state={'schemaVersion':1,'current':OLD,'previous':None,'runningRevision':OLD,'bootstrapVerified':True,'status':'active'}
        self.write_state()
        (self.root/'current').symlink_to('releases/'+OLD)

    def write_state(self):
        self.state_path.write_text(json.dumps(self.state));self.state_path.chmod(0o600)

    def implementation(self):
        self.assertIsNotNone(module,'deployment state machine is missing')
        return module

    def transport(self,root,revision,args,sudo=False,acme='production'):
        self.commands.append((revision,args,acme))
        stdout=b'a'*64+b'\n' if args==['ps','-q','db'] else (json.dumps({'status':'ready','topic':{'taxonomy':True,'study':True,'retirement':True,'schemaReady':True,'topicsMode':False},'content':{'capability':True,'schemaReady':True,'managedMode':False}}).encode() if args[:4]==['exec','-T','api','wget'] else b'')
        return subprocess.CompletedProcess(args,0,stdout,b'')

    def invoke(self,*args):
        output=io.StringIO()
        with redirect_stdout(output),redirect_stderr(output):
            status=self.implementation().main([*args,'--root',str(self.root)])
        return status,output.getvalue()

    def test_invalid_release_does_not_mutate(self):
        deploy=self.implementation()
        before=self.state_path.read_bytes()
        with patch.object(common,'compose',side_effect=AssertionError('unexpected Docker mutation')):
            for value in ['../unsafe','z'*40]:
                self.assertEqual(self.invoke('prepare','--revision',value)[0],1)
            alias=self.root/'alias';alias.symlink_to(self.root)
            self.assertEqual(deploy.main(['prepare','--root',str(alias),'--revision',NEW]),1)
            with (self.root/'shared'/'deployment.lock').open('w') as lock:
                fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
                self.assertEqual(self.invoke('prepare','--revision',NEW)[0],1)
        self.assertEqual(self.state_path.read_bytes(),before)
        self.assertEqual((self.root/'current').readlink().as_posix(),'releases/'+OLD)

    def test_failed_health_keeps_current_release(self):
        deploy=self.implementation()
        def transport(root,revision,args,sudo=False,acme='production'):
            result=self.transport(root,revision,args,sudo,acme)
            if revision==NEW and args[:2]==['up','-d'] and 'web' in args:
                result.returncode=1
            return result
        with patch.object(common,'compose',side_effect=transport),patch.object(deploy.snapshot,'capture',return_value={'ok':True,'tables':1,'workspaces':0,'dumpSha256':'b'*64}),patch.object(deploy.snapshot,'sql',return_value=str(SCHEMA)),patch.object(deploy,'trusted_check',return_value={'ok':True}):
            self.assertEqual(self.invoke('start','--revision',NEW)[0],1)
        current=json.loads(self.state_path.read_text())
        self.assertEqual(current['current'],OLD)
        self.assertEqual(current['runningRevision'],OLD)
        self.assertEqual((self.root/'current').readlink().as_posix(),'releases/'+OLD)
        self.assertTrue(any(revision==OLD and 'web' in args for revision,args,_ in self.commands))

    def test_rollback_keeps_database_and_cert_volumes(self):
        deploy=self.implementation()
        self.state.update(current=NEW,previous=OLD,runningRevision=NEW);self.write_state()
        (self.root/'current').unlink();(self.root/'current').symlink_to('releases/'+NEW)
        with patch.object(common,'compose',side_effect=self.transport),patch.object(deploy.snapshot,'sql',return_value=str(SCHEMA)),patch.object(deploy,'trusted_check',return_value={'ok':True}):
            self.assertEqual(self.invoke('rollback','--revision',OLD)[0],0)
        state=json.loads(self.state_path.read_text())
        self.assertEqual(state['current'],OLD)
        self.assertEqual((self.root/'current').readlink().as_posix(),'releases/'+OLD)
        for _,args,_ in self.commands:
            self.assertNotIn('down',args)
            self.assertNotIn('-v',args)
            self.assertNotIn('down -v',args)

    def first_deploy_fixture(self):
        self.state.update(current=None,previous=None,runningRevision=None,bootstrapVerified=False)
        self.write_state()
        if (self.root/'current').is_symlink():(self.root/'current').unlink()
        return {'sourceCommit':OLD,'dumpSha256':'b'*64,'migrationVersion':SCHEMA}

    def test_first_deploy_retry_retains_database_provenance(self):
        deploy=self.implementation()
        for fault in ['backup','health','activate']:
            with self.subTest(fault=fault):
                baseline=self.first_deploy_fixture()
                def transport(root,revision,args,sudo=False,acme='production'):
                    result=self.transport(root,revision,args,sudo,acme)
                    if fault=='health' and args[:2]==['up','-d'] and 'web' in args:result.returncode=1
                    return result
                captures=[deploy.snapshot.SnapshotError('backup-failed')] if fault=='backup' else [{'ok':True}]
                with patch.object(common,'compose',side_effect=transport),patch.object(deploy.snapshot,'load_backup',return_value=baseline),patch.object(deploy.snapshot,'inspect',return_value={'ok':True}),patch.object(deploy.snapshot,'capture',side_effect=captures),patch.object(deploy,'trusted_check',side_effect=deploy.DeployError('trusted-https-check-failed')):
                    status,_=self.invoke('start','--revision',NEW,'--baseline',str(self.root/'baseline'))
                    if fault=='activate':
                        self.assertEqual(status,0)
                        self.assertEqual(self.invoke('activate','--revision',NEW,'--acme','production')[0],1)
                    else:self.assertEqual(status,1)
                with patch.object(common,'compose',side_effect=self.transport),patch.object(deploy.snapshot,'capture',return_value={'ok':True}):
                    status,_=self.invoke('start','--revision',NEW)
                self.assertEqual(status,0,'verified first-deploy baseline must support retry')
                self.assertEqual(json.loads(self.state_path.read_text())['runningRevision'],NEW)

    def test_explicit_rollback_failure_stops_and_records(self):
        deploy=self.implementation()
        for fault in ['health','tls']:
            with self.subTest(fault=fault):
                self.commands=[]
                self.state.update(current=NEW,previous=OLD,runningRevision=NEW);self.write_state()
                def transport(root,revision,args,sudo=False,acme='production'):
                    result=self.transport(root,revision,args,sudo,acme)
                    if fault=='health' and args[:2]==['up','-d'] and 'web' in args:result.returncode=1
                    return result
                checker=patch.object(deploy,'trusted_check',side_effect=deploy.DeployError('trusted-https-check-failed')) if fault=='tls' else patch.object(deploy,'trusted_check',return_value={'ok':True})
                with patch.object(common,'compose',side_effect=transport),patch.object(deploy.snapshot,'sql',return_value=str(SCHEMA)),checker:
                    self.assertEqual(self.invoke('rollback','--revision',OLD)[0],1)
                state=json.loads(self.state_path.read_text())
                self.assertEqual(state['current'],NEW)
                self.assertEqual(state['status'],'manual-recovery-required')
                self.assertTrue(any(args[:1]==['stop'] and all(v in args for v in ['gateway','web','api']) for _,args,_ in self.commands))

    def test_start_stops_public_writes_before_migration(self):
        deploy=self.implementation()
        with patch.object(common,'compose',side_effect=self.transport),patch.object(deploy.snapshot,'capture',return_value={'ok':True}):
            self.assertEqual(self.invoke('start','--revision',NEW)[0],0)
        stopped=[i for i,(_,args,_) in enumerate(self.commands) if args[:1]==['stop'] and all(v in args for v in ['gateway','web','api'])]
        migrated=[i for i,(_,args,_) in enumerate(self.commands) if '/app/bin/migrate' in args]
        self.assertTrue(stopped,'public traffic must stop before schema migration')
        self.assertLess(stopped[0],migrated[0])
        self.assertFalse(any(args[:1]==['up'] and 'gateway' in args for _,args,_ in self.commands))

    def test_secrets_are_not_logged(self):
        deploy=self.implementation()
        secret='test-secret-that-must-not-appear'
        def failure(root,revision,args,sudo=False,acme='production'):
            self.commands.append((revision,args,acme))
            return subprocess.CompletedProcess(args,1,secret.encode(),('postgres://user:'+secret+'@db').encode())
        with patch.object(common,'compose',side_effect=failure):
            status,output=self.invoke('prepare','--revision',NEW)
        self.assertEqual(status,1)
        self.assertNotIn(secret,output)
        self.assertNotIn('postgres://',output)
        self.assertFalse(json.loads(output.strip().splitlines()[-1])['ok'])

    def test_schema_mismatch_stops_writes_instead_of_rollback(self):
        deploy=self.implementation()
        self.state.update(current=NEW,previous=OLD,runningRevision=NEW);self.write_state()
        with patch.object(common,'compose',side_effect=self.transport),patch.object(deploy.snapshot,'sql',return_value=str(SCHEMA+1)):
            self.assertEqual(self.invoke('rollback','--revision',OLD)[0],1)
        self.assertTrue(any(args[:1]==['stop'] for _,args,_ in self.commands))
        self.assertFalse(any(revision==OLD and args[:1]==['up'] for revision,args,_ in self.commands))

    def test_topic_old_binary_keeps_gateway_isolated(self):
        deploy=self.implementation();self.state.update(current=NEW,previous=OLD,runningRevision=NEW);self.write_state();marker=self.root/'shared'/'database.marker';marker.write_bytes(b'original database fingerprint')
        before=marker.read_bytes()
        with patch.object(common,'compose',side_effect=self.transport),patch.object(deploy.snapshot,'sql',return_value=str(SCHEMA)),patch.object(deploy,'topic_database_state',return_value={'requiredMigration':12,'mode':'topics'}),patch.object(deploy,'runtime_topic_health',return_value={'taxonomy':True,'study':True,'retirement':False,'schemaReady':True,'topicsMode':True}):
            self.assertEqual(self.invoke('rollback','--revision',OLD)[0],1)
        self.assertEqual(marker.read_bytes(),before)
        self.assertFalse(any(args[:2]==['up','-d'] and 'gateway' in args for _,args,_ in self.commands))
        self.assertEqual(json.loads(self.state_path.read_text())['status'],'manual-recovery-required')
    def test_topic_partial_schema_keeps_maintenance(self):
        deploy=self.implementation()
        with patch.object(common,'compose',side_effect=self.transport),patch.object(deploy.snapshot,'sql',return_value=str(SCHEMA)),patch.object(deploy,'runtime_topic_health',return_value={'taxonomy':True,'study':True,'retirement':True,'schemaReady':False,'topicsMode':False}):
            self.assertEqual(self.invoke('rollback','--revision',OLD)[0],1)
        self.assertFalse(any(args[:2]==['up','-d'] and 'gateway' in args for _,args,_ in self.commands))

class TopicCompatibilityTests(unittest.TestCase):
    def test_topic_mode_old_binary_rejected(self):
        self.assertFalse(module.topicSchemaCompatibility(12,{'taxonomy':True,'study':True,'retirement':False},'topics'))
        self.assertFalse(module.topicSchemaCompatibility(13,{'taxonomy':True,'study':True,'retirement':True},'topics'))
    def test_topic_schema_partial_keeps_maintenance(self):
        self.assertFalse(module.topicSchemaCompatibility(11,{'taxonomy':True,'study':True,'retirement':True},'topics'))
        self.assertFalse(module.topicSchemaCompatibility(12,{'taxonomy':True,'study':False,'retirement':True},'legacy'))
    def test_compatible_binary_keeps_database(self):
        self.assertTrue(module.topicSchemaCompatibility(12,{'taxonomy':True,'study':True,'retirement':True},'topics'))
        self.assertTrue(module.topicSchemaCompatibility(9,{'taxonomy':False,'study':False,'retirement':False},'legacy'))


if __name__=="__main__":unittest.main()

class ManagedCompatibilityTests(unittest.TestCase):
    def test_thirteen_requires_the_explicit_managed_binary_capability(self):
        base={'taxonomy':True,'study':True,'retirement':True}
        self.assertFalse(module.topicSchemaCompatibility(13,base,'topics'))
        self.assertTrue(module.topicSchemaCompatibility(13,{**base,'managed':True},'topics'))
        self.assertFalse(module.topicSchemaCompatibility(13,{**base,'managed':False},'topics'))
