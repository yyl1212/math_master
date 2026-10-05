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
        stdout=b'a'*64+b'\n' if args==['ps','-q','db'] else b''
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
        with patch.object(common,'compose',side_effect=transport),patch.object(deploy.snapshot,'capture',return_value={'ok':True,'tables':1,'workspaces':0,'dumpSha256':'b'*64}),patch.object(deploy.snapshot,'sql',return_value='8'),patch.object(deploy,'trusted_check',return_value={'ok':True}):
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
        with patch.object(common,'compose',side_effect=self.transport),patch.object(deploy.snapshot,'sql',return_value='8'),patch.object(deploy,'trusted_check',return_value={'ok':True}):
            self.assertEqual(self.invoke('rollback','--revision',OLD)[0],0)
        state=json.loads(self.state_path.read_text())
        self.assertEqual(state['current'],OLD)
        self.assertEqual((self.root/'current').readlink().as_posix(),'releases/'+OLD)
        for _,args,_ in self.commands:
            self.assertNotIn('down',args)
            self.assertNotIn('-v',args)
            self.assertNotIn('down -v',args)

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
        with patch.object(common,'compose',side_effect=self.transport),patch.object(deploy.snapshot,'sql',return_value='9'):
            self.assertEqual(self.invoke('rollback','--revision',OLD)[0],1)
        self.assertTrue(any(args[:1]==['stop'] for _,args,_ in self.commands))
        self.assertFalse(any(revision==OLD and args[:1]==['up'] for revision,args,_ in self.commands))


if __name__=='__main__':unittest.main()
