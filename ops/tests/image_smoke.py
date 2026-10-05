#!/usr/bin/env python3
"""真实镜像契约烟测；只清理本次创建的临时容器。"""
import argparse
import json
import os
from pathlib import Path
import shutil
import re
import subprocess
import sys
import tempfile
import time
import uuid


OPS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(OPS))
import common


def runtime_mounts(revision: str) -> dict[str, list[str]]:
    # 解析生产 Compose，容器烟测必须使用真实挂载配置。
    with tempfile.TemporaryDirectory(dir='/private/tmp' if Path('/private/tmp').exists() else None) as folder:
        root = Path(folder)
        config_path = root / 'shared' / 'configs' / (revision + '.env')
        config_path.parent.mkdir(parents=True, mode=0o700)
        values = {'AUTH_PUBLIC_ORIGIN': common.ORIGIN, 'POSTGRES_DB': 'math_master_preview',
                  'POSTGRES_USER': 'math_master_preview', 'POSTGRES_PASSWORD': 'a' * 64,
                  'DB_LC_COLLATE': 'en_US.utf8', 'DB_LC_CTYPE': 'en_US.utf8'}
        descriptor = os.open(config_path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(descriptor, 'w') as stream:
            stream.write(''.join(key + '=' + value + '\n' for key, value in values.items()))
        release = root / 'releases' / revision
        release.mkdir(parents=True)
        shutil.copyfile(OPS.parent / 'compose.yaml', release / 'compose.yaml')
        result = common.compose(root, revision, ['config', '--format', 'json'])
        if result.returncode:
            raise RuntimeError('runtime-compose-invalid')
        config = json.loads(result.stdout)
        return {part: [argument for mount in config['services'][part]['tmpfs']
                       for argument in ['--tmpfs', mount]] for part in ['api', 'web']}


def docker(*args: str) -> bytes:
    result = subprocess.run(['docker', *args], capture_output=True, timeout=120)
    if result.returncode:
        raise RuntimeError('docker-operation-failed')
    return result.stdout


def check(revision: str) -> dict:
    images = {part: f'math-master-{part}:{revision}' for part in ['api', 'web', 'gateway']}
    for part, image in images.items():
        try:
            label = docker('image', 'inspect', '--format', '{{index .Config.Labels "org.opencontainers.image.revision"}}', image).decode().strip()
        except RuntimeError:
            raise RuntimeError(f'{part}-image-missing') from None
        if label != revision:
            raise RuntimeError(f'{part}-revision-mismatch')
    mounts = runtime_mounts(revision)
    docker('run', '--rm', '--network', 'none', '--read-only', *mounts['api'], '--entrypoint', 'sh', images['api'], '-ec',
           'test "$(id -u)" -ne 0; for name in server migrate admin-init correction-maintenance; do test -x /app/bin/$name; done; test ! -e /app/bin/e2e-harness; test -f /app/db/migrations/00008_correction_notifications.sql')
    version = docker('run', '--rm', '--network', 'none', '--read-only', images['gateway'], 'version').decode().strip()
    if not version.startswith('v2.11.7 '):
        raise RuntimeError('gateway-version-mismatch')
    name = f'math-master-image-smoke-{uuid.uuid4().hex}'
    created = False
    try:
        docker('run', '--detach', '--name', name, '--network', 'none', '--read-only', *mounts['web'],
               '--env', 'APP_ENV=production', '--env', 'AUTH_PUBLIC_ORIGIN=https://43.135.142.53',
               '--env', 'GO_API_INTERNAL_URL=http://api:8080', '--env', 'HOSTNAME=0.0.0.0', '--env', 'PORT=3000', images['web'])
        created = True
        probe = r'''
const origin = 'http://127.0.0.1:3000';
const page = await fetch(origin + '/login', {signal: AbortSignal.timeout(10000)});
if (page.status !== 200) throw Error('login-status');
const html = await page.text();
const css = [...html.matchAll(/href="([^\"]+\.css(?:\?[^\"]*)?)"/g)].map(m => m[1].replaceAll('&amp;', '&'));
if (!css.length) throw Error('missing-css-links');
const fonts = new Set();
for (const href of new Set(css)) {
  const url = new URL(href, origin);
  const response = await fetch(url, {signal: AbortSignal.timeout(5000)});
  if (response.status !== 200) throw Error('css-status');
  const text = await response.text();
  for (const m of text.matchAll(/url\(["']?([^\s)"']+)["']?\)/g)) {
    if (/KaTeX[^/]*\.woff2(?:\?|$)/i.test(m[1])) fonts.add(new URL(m[1], url).href);
  }
}
if (!fonts.size) throw Error('missing-katex-fonts');
for (const font of fonts) {
  const response = await fetch(font, {signal: AbortSignal.timeout(5000)});
  if (response.status !== 200 || !(await response.arrayBuffer()).byteLength) throw Error('font-status');
}
console.log(JSON.stringify({login: true, stylesheets: new Set(css).size, katexFonts: fonts.size}));
'''
        result = None
        for _ in range(20):
            try:
                result = json.loads(docker('exec', name, 'node', '--input-type=module', '-e', probe))
                break
            except RuntimeError:
                time.sleep(0.5)
        if not result:
            raise RuntimeError('web-assets-unavailable')
        return {'ok': True, 'revision': revision, 'apiWhitelist': True, 'harnessAbsent': True, 'gatewayVersion': version.split()[0], **result}
    finally:
        if created:
            docker('rm', '--force', name)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--revision', required=True)
    args = parser.parse_args()
    if not re.fullmatch(r'[0-9a-f]{40}', args.revision):
        parser.error('revision must be a 40-character lowercase commit')
    try:
        print(json.dumps(check(args.revision)))
        return 0
    except (RuntimeError, subprocess.SubprocessError) as error:
        stage = str(error) if re.fullmatch(r'[a-z-]+', str(error)) else 'image-smoke-failed'
        print(json.dumps({'ok': False, 'stage': stage}), file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
