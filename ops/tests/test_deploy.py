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
        public_health=patch.object(module,'verify_note_sharing_public',return_value=None)
        public_health.start();self.addCleanup(public_health.stop)
        review_health=patch.object(module,'verify_study_review_public',return_value=None)
        review_health.start();self.addCleanup(review_health.stop)
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
        stdout=b'a'*64+b'\n' if args==['ps','-q','db'] else (b'{"capability":true,"schemaReady":true}' if args and args[-1].endswith(('/api/v3/note-sharing/health','/api/v3/study/review/health')) else json.dumps({'status':'ready','topic':{'taxonomy':True,'study':True,'retirement':True,'schemaReady':True,'topicsMode':False},'content':{'capability':True,'schemaReady':True,'managedMode':False}}).encode() if args[:4]==['exec','-T','api','wget'] else b'')
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

    def test_old_fourteen_release_cannot_rollback_fifteen_database(self):
        (self.root/'releases'/OLD/'db'/'migrations'/'00015_study_review.sql').unlink()
        self.state.update(current=NEW,previous=OLD,runningRevision=NEW);self.write_state()
        with patch.object(common,'compose',side_effect=self.transport),patch.object(module.snapshot,'sql',return_value='15'):
            self.assertEqual(self.invoke('rollback','--revision',OLD)[0],1)
        self.assertFalse(any(args[:2]==['up','-d'] and any(v in args for v in ['api','web','gateway']) for _,args,_ in self.commands))
        self.assertFalse(any(args and args[0]=='down' for _,args,_ in self.commands))
        self.assertEqual(json.loads(self.state_path.read_text())['status'],'manual-recovery-required')

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

class NoteSharingCompatibilityTests(unittest.TestCase):
    def test_fourteen_requires_note_sharing_health(self):
        base={'taxonomy':True,'study':True,'retirement':True,'managed':True}
        self.assertFalse(module.topicSchemaCompatibility(14,base,'topics'))
        self.assertFalse(module.topicSchemaCompatibility(14,{**base,'noteSharing':False},'topics'))
        self.assertTrue(module.topicSchemaCompatibility(14,{**base,'noteSharing':True},'topics'))
        self.assertTrue(module.topicSchemaCompatibility(13,base,'topics'))
    def test_future_migration_rejected(self):
        self.assertFalse(module.topicSchemaCompatibility(15,{'taxonomy':True,'study':True,'retirement':True,'managed':True,'noteSharing':True},'topics'))
    def test_old_managed_binary_cannot_serve_fourteen(self):
        with patch.object(module,'topic_database_state',return_value={'requiredMigration':14,'mode':'topics','managedMode':True}),patch.object(module,'runtime_topic_health',return_value={'taxonomy':True,'study':True,'retirement':True,'schemaReady':True,'topicsMode':True,'managed':True,'managedMode':True}),patch.object(module,'runtime_note_sharing_health',side_effect=module.DeployError('note-sharing-readiness-unavailable')):
            with self.assertRaises(module.DeployError):module.verify_topic_runtime(Path('/private/tmp'),NEW,False,'container')
    def test_note_sharing_health_is_strict_and_bounded(self):
        for payload in [{'capability':True,'schemaReady':False},{'capability':True,'schemaReady':True,'body':'private'},{'capability':'true','schemaReady':True}]:
            with patch.object(module,'run_compose',return_value=json.dumps(payload).encode()):
                with self.assertRaises(module.DeployError):module.runtime_note_sharing_health(Path('/private/tmp'),NEW,False)
        with patch.object(module,'run_compose',return_value=b'{"capability":true,"schemaReady":true}'):
            self.assertEqual(module.runtime_note_sharing_health(Path('/private/tmp'),NEW,False),{'capability':True,'schemaReady':True})
    def test_start_and_activate_probe_sharing(self):
        order=[];state={'bootstrapVerified':True,'runningRevision':OLD,'current':OLD}
        def compose(root,revision,args,sudo=False,failure=''):
            order.append('web' if args[:2]==['up','-d'] and 'web' in args else 'api' if args[:2]==['up','-d'] and 'api' in args else 'other');return b''
        with patch.object(module,'database_container',return_value='container'),patch.object(module,'stop_writes'),patch.object(module.snapshot,'capture'),patch.object(module,'run_compose',side_effect=compose),patch.object(module,'verify_topic_runtime',side_effect=lambda *a:order.append('go-probe')),patch.object(module,'verify_note_sharing_next',side_effect=lambda *a:order.append('next-probe')),patch.object(module,'verify_study_review_next'),patch.object(module,'save_state'):
            module.start(Path('/private/tmp'),NEW,False,state,None)
        self.assertLess(order.index('go-probe'),order.index('web'));self.assertLess(order.index('web'),order.index('next-probe'))
        order=[];state={'bootstrapVerified':True,'runningRevision':NEW}
        with patch.object(module,'database_container',return_value='container'),patch.object(module,'verify_topic_runtime',side_effect=lambda *a:order.append('go-probe')),patch.object(module,'run_compose',side_effect=lambda *a:order.append('gateway')),patch.object(module,'trusted_check',side_effect=lambda *a:order.append('https')),patch.object(module,'verify_note_sharing_public',side_effect=lambda *a:order.append('public-probe')),patch.object(module,'verify_study_review_public'),patch.object(module,'publish_current'):
            module.activate(Path('/private/tmp'),NEW,False,state,'production')
        self.assertLess(order.index('gateway'),order.index('public-probe'))

class StudyReviewCompatibilityTests(unittest.TestCase):
    capabilities={'taxonomy':True,'study':True,'retirement':True,'managed':True,'noteSharing':True,'studyReview':True}
    def test_fifteen_requires_explicit_review_and_future_sixteen_is_rejected(self):
        self.assertTrue(module.topicSchemaCompatibility(15,self.capabilities,'topics'))
        for key in self.capabilities:
            missing={**self.capabilities,key:False}
            self.assertFalse(module.topicSchemaCompatibility(15,missing,'topics'))
        self.assertFalse(module.topicSchemaCompatibility(16,self.capabilities,'topics'))
        self.assertFalse(module.topicSchemaCompatibility(15,{k:v for k,v in self.capabilities.items() if k!='studyReview'},'topics'))
        self.assertFalse(module.topicSchemaCompatibility(15,{**self.capabilities,'studyReview':1},'topics'))
    def test_review_health_is_private_body_free_strict_and_bounded(self):
        for payload in [b'{}',b'[]',b'null',b'bad-json',b'x'*2049,b'{"capability":true,"schemaReady":false}',b'{"capability":"true","schemaReady":true}',b'{"capability":true,"schemaReady":true,"answer":"secret"}']:
            with self.assertRaises((ValueError,TypeError)):module.strict_study_review_health(payload)
        self.assertEqual(module.strict_study_review_health(b'{"capability":true,"schemaReady":true}'),{'capability':True,'schemaReady':True})
    def test_go_missing_review_stops_fifteen(self):
        base={'taxonomy':True,'study':True,'retirement':True,'schemaReady':True,'topicsMode':True,'managed':True,'managedMode':True}
        with patch.object(module,'topic_database_state',return_value={'requiredMigration':15,'mode':'topics','managedMode':True}),patch.object(module,'runtime_topic_health',return_value=base),patch.object(module,'runtime_note_sharing_health',return_value={'capability':True,'schemaReady':True}),patch.object(module,'runtime_study_review_health',side_effect=module.DeployError('study-review-readiness-unavailable')):
            with self.assertRaises(module.DeployError):module.verify_topic_runtime(Path('/private/tmp'),NEW,False,'container')
    def test_next_probe_and_original_fourteen_boundaries(self):
        with patch.object(module,'expected_migration',return_value=14),patch.object(module,'run_compose',side_effect=AssertionError('old release probed new endpoint')):
            module.verify_study_review_next(Path('/private/tmp'),OLD,False)
        with patch.object(module,'expected_migration',return_value=15),patch.object(module,'run_compose',return_value=b'{"capability":true,"schemaReady":true}') as request:
            module.verify_study_review_next(Path('/private/tmp'),NEW,False)
            self.assertEqual(request.call_args.args[2][-1],'http://127.0.0.1:3000/api/v3/study/review/health')
        with patch.object(module,'expected_migration',return_value=15),patch.object(module,'run_compose',return_value=b'{}'):
            with self.assertRaises(module.DeployError):module.verify_study_review_next(Path('/private/tmp'),NEW,False)
    def test_public_probe_is_https_redirect_free_and_rejects_missing_health(self):
        from unittest.mock import MagicMock
        import urllib.request
        url=common.ORIGIN+'/api/v3/study/review/health'
        for valid,status,target,payload in [(True,200,url,b'{"capability":true,"schemaReady":true}'),(False,404,url,b'{}'),(False,200,url+'/redirect',b'{"capability":true,"schemaReady":true}'),(False,200,url,b'{"capability":true,"schemaReady":false}')]:
            response=MagicMock();response.status=status;response.geturl.return_value=target;response.read.return_value=payload;response.__enter__.return_value=response
            with patch.object(module,'expected_migration',return_value=15),patch.object(urllib.request,'urlopen',return_value=response) as request:
                if valid:
                    module.verify_study_review_public(Path('/private/tmp'),NEW)
                    response.read.assert_called_once_with(2049)
                    self.assertEqual(request.call_args.kwargs['timeout'],5)
                else:
                    with self.assertRaises(module.DeployError):module.verify_study_review_public(Path('/private/tmp'),NEW)
    def test_review_probes_run_before_ready_and_publish(self):
        order=[];state={'bootstrapVerified':True,'runningRevision':OLD,'current':OLD}
        def compose(root,revision,args,sudo=False,failure=''):
            order.append('web' if args[:2]==['up','-d'] and 'web' in args else 'api' if args[:2]==['up','-d'] and 'api' in args else 'other');return b''
        with patch.object(module,'database_container',return_value='container'),patch.object(module,'stop_writes'),patch.object(module.snapshot,'capture'),patch.object(module,'run_compose',side_effect=compose),patch.object(module,'verify_topic_runtime',side_effect=lambda *a:order.append('go-review')),patch.object(module,'verify_note_sharing_next'),patch.object(module,'verify_study_review_next',side_effect=lambda *a:order.append('next-review')),patch.object(module,'save_state',side_effect=lambda *a:order.append('ready')):
            module.start(Path('/private/tmp'),NEW,False,state,None)
        self.assertLess(order.index('api'),order.index('go-review'));self.assertLess(order.index('web'),order.index('next-review'));self.assertLess(order.index('next-review'),order.index('ready'))
        order=[];state['runningRevision']=NEW
        with patch.object(module,'database_container',return_value='container'),patch.object(module,'verify_topic_runtime',side_effect=lambda *a:order.append('go-review')),patch.object(module,'run_compose',side_effect=lambda *a:order.append('gateway')),patch.object(module,'trusted_check',side_effect=lambda *a:order.append('https')),patch.object(module,'verify_note_sharing_public'),patch.object(module,'verify_study_review_public',side_effect=lambda *a:order.append('public-review')),patch.object(module,'publish_current',side_effect=lambda *a:order.append('published')):
            module.activate(Path('/private/tmp'),NEW,False,state,'production')
        self.assertLess(order.index('gateway'),order.index('public-review'));self.assertLess(order.index('public-review'),order.index('published'))
