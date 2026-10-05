"""配置边界测试：阻止命令执行、生产/开发漂移与意外端口暴露。"""
import json
import os
from pathlib import Path
import shutil
import sys
import tempfile
import unittest
from unittest.mock import patch

OPS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(OPS))
try:
    import common
except ModuleNotFoundError:
    common = None

REVISION = '1' * 40
BODY = '\n'.join([
    'AUTH_PUBLIC_ORIGIN=https://43.135.142.53',
    'POSTGRES_DB=math_master_preview', 'POSTGRES_USER=math_master_preview',
    'POSTGRES_PASSWORD=' + 'a' * 64,
    'DB_LC_COLLATE=en_US.utf8', 'DB_LC_CTYPE=en_US.utf8', '',
])


class ConfigTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(dir='/private/tmp' if Path('/private/tmp').exists() else None)
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.env = self.root / 'shared' / 'configs' / (REVISION + '.env')
        self.env.parent.mkdir(parents=True, mode=0o700)
        self.write_env(BODY)
        release = self.root / 'releases' / REVISION
        release.mkdir(parents=True)
        if (OPS.parent / 'compose.yaml').exists():
            shutil.copyfile(OPS.parent / 'compose.yaml', release / 'compose.yaml')
        (release / 'ops').mkdir()
        if (OPS / 'Caddyfile').exists():
            shutil.copyfile(OPS / 'Caddyfile', release / 'ops' / 'Caddyfile')

    def implementation(self):
        self.assertIsNotNone(common, 'production configuration validation is missing')
        return common

    def write_env(self, body):
        self.env.write_text(body)
        self.env.chmod(0o600)

    def test_rejects_executable_or_insecure_env(self):
        module = self.implementation()
        sentinel = self.root / 'executed'
        variants = [
            BODY + f'EXTRA=$(touch {sentinel})\n',
            BODY.replace('en_US.utf8', f'`touch {sentinel}`'),
            BODY + 'POSTGRES_DB=math_master_preview\n',
            BODY.replace('https://43.135.142.53', 'http://43.135.142.53'),
        ]
        for body in variants:
            with self.subTest(kind=variants.index(body)):
                self.write_env(body)
                with patch.object(module.subprocess, 'run', side_effect=AssertionError('unexpected Docker mutation')):
                    with self.assertRaises(ValueError):
                        module.compose(self.root, REVISION, ['up', '-d', 'db'])
                self.assertFalse(sentinel.exists())
        self.write_env(BODY)
        self.env.chmod(0o644)
        with self.assertRaises(ValueError):
            module.load_env(self.env)
        self.env.chmod(0o600)
        target = self.env.with_suffix('.target')
        self.env.rename(target)
        self.env.symlink_to(target)
        with self.assertRaises(ValueError):
            module.load_env(self.env)

    def test_ignores_inherited_development_settings(self):
        module = self.implementation()
        with patch.dict(os.environ, {'APP_ENV': 'development', 'DATABASE_URL': 'invalid-dev-url', 'POSTGRES_PASSWORD': 'inherited-wrong-password', 'DOCKER_HOST': 'tcp://unsafe.invalid:2375'}):
            result = module.compose(self.root, REVISION, ['config', '--format', 'json'])
        self.assertEqual(result.returncode, 0, 'Compose configuration did not validate')
        config = json.loads(result.stdout)
        for service in ['api', 'web']:
            self.assertEqual(config['services'][service]['environment']['APP_ENV'], 'production')
            self.assertEqual(config['services'][service]['environment']['AUTH_PUBLIC_ORIGIN'], 'https://43.135.142.53')
        self.assertNotIn('invalid-dev-url', config['services']['api']['environment']['DATABASE_URL'])
        self.assertNotIn('inherited-wrong-password', config['services']['api']['environment']['DATABASE_URL'])

    def test_application_tmpfs_mounts_are_absolute_and_bounded(self):
        result = self.implementation().compose(self.root, REVISION, ['config', '--format', 'json'])
        self.assertEqual(result.returncode, 0)
        config = json.loads(result.stdout)
        for service in ['api', 'web']:
            with self.subTest(service=service):
                mounts = config['services'][service]['tmpfs']
                self.assertTrue(all(mount.split(':', 1)[0].startswith('/') for mount in mounts),
                                'Docker rejects relative tmpfs mount destinations')
                self.assertIn('/tmp:rw,size=64m', mounts)
        self.assertIn('/app/.next/cache:rw,size=64m,uid=1000,gid=1000,mode=0700',
                      config['services']['web']['tmpfs'])

    def test_only_gateway_publishes_ports(self):
        module = self.implementation()
        result = module.compose(self.root, REVISION, ['config', '--format', 'json'])
        self.assertEqual(result.returncode, 0)
        config = json.loads(result.stdout)
        for service in ['api', 'web', 'db']:
            self.assertFalse(config['services'][service].get('ports'))
        ports = config['services']['gateway']['ports']
        self.assertEqual({(str(p['published']), p['target']) for p in ports}, {('80', 80), ('443', 443)})
        self.assertEqual(config['volumes']['postgres-data']['name'], 'math-master-preview-pgdata')
        staging = module.compose(self.root, REVISION, ['config', '--format', 'json'], acme='staging')
        stage_config = json.loads(staging.stdout)
        self.assertEqual(stage_config['volumes']['caddy-data']['name'], 'math-master-preview-caddy-staging')
        self.assertEqual(config['volumes']['caddy-data']['name'], 'math-master-preview-caddy-production')


if __name__ == '__main__':
    unittest.main()
