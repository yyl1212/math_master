#!/usr/bin/env python3
"""真实镜像契约烟测；只清理本次创建的临时容器。"""
import argparse
import json
import os
from pathlib import Path
import shutil
import socket
import ssl
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


def check_ip_tls(image: str, web_image: str) -> dict:
    # 本机隔离夹具仅显式信任一次性测试证书；生产验收仍使用系统 CA。
    with tempfile.TemporaryDirectory(dir='/private/tmp' if Path('/private/tmp').exists() else None) as folder:
        root = Path(folder)
        fixture = root / 'fixture'
        fixture.mkdir(mode=0o755)
        fixture.chmod(0o755)
        certificate, key = fixture / 'server.crt', fixture / 'server.key'
        subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-days', '1',
                        '-subj', '/CN=43.135.142.53', '-addext', 'subjectAltName=IP:43.135.142.53',
                        '-out', str(certificate), '-keyout', str(key)],
                       check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15)
        # root 为 0700；仅容器内非 root Caddy 读取这些一次性夹具文件。
        certificate.chmod(0o644); key.chmod(0o644)
        config = json.loads(docker('run', '--rm', '--network', 'none', '--read-only',
                           '--env', 'PUBLIC_HOST=43.135.142.53',
                           '--env', 'ACME_DIRECTORY=https://acme-staging-v02.api.letsencrypt.org/directory',
                           '--mount', f'type=bind,src={OPS / "Caddyfile"},dst=/etc/caddy/Caddyfile,readonly',
                           image, 'adapt', '--config', '/etc/caddy/Caddyfile', '--adapter', 'caddyfile'))
        config['apps']['tls'] = {'certificates': {'load_files': [
            {'certificate': '/fixture/server.crt', 'key': '/fixture/server.key'}]}}
        for server in config['apps']['http']['servers'].values():
            server['automatic_https'] = {'disable': True}
        (fixture / 'config.json').write_text(json.dumps(config))
        (fixture / 'config.json').chmod(0o644)
        token = uuid.uuid4().hex
        name = 'math-master-ip-tls-smoke-' + token
        upstream = 'math-master-route-upstream-' + token
        network = 'math-master-route-network-' + token
        created = upstream_created = network_created = False
        try:
            docker('network', 'create', '--internal', network)
            network_created = True
            probe = r'''
const http = require('node:http');
for (const [port, source] of [[8080, 'api'], [3000, 'web']]) {
  http.createServer((request, response) => {
    response.setHeader('Content-Type', 'text/plain');
    response.end(source + ':' + request.url);
  }).listen(port, '0.0.0.0');
}
'''
            docker('run', '--detach', '--name', upstream, '--network', network,
                   '--network-alias', 'api', '--network-alias', 'web', '--read-only',
                   '--entrypoint', 'node', web_image, '-e', probe)
            upstream_created = True
            docker('run', '--detach', '--name', name, '--read-only',
                   '--network', 'bridge',
                   '--sysctl', 'net.ipv4.ip_unprivileged_port_start=0',
                   '--tmpfs', '/data:rw,size=16m,uid=10001,gid=10001,mode=0700',
                   '--tmpfs', '/config:rw,size=16m,uid=10001,gid=10001,mode=0700',
                   '--publish', '127.0.0.1::443',
                   '--mount', f'type=bind,src={fixture},dst=/fixture,readonly',
                   image, 'run', '--config', '/fixture/config.json')
            created = True
            docker('network', 'connect', network, name)
            ready = False
            for _ in range(20):
                try:
                    docker('exec', name, 'wget', '-q', '-O', '/dev/null', 'http://127.0.0.1:2019/config/')
                    ready = True; break
                except RuntimeError:
                    time.sleep(0.25)
            if not ready:raise RuntimeError('ip-tls-gateway-not-ready')
            ports = json.loads(docker('inspect', '--format', '{{json .NetworkSettings.Ports}}', name))
            port = int(ports['443/tcp'][0]['HostPort'])
            def handshake(context, hostname):
                with socket.create_connection(('127.0.0.1', port), timeout=5) as transport:
                    with context.wrap_socket(transport, server_hostname=hostname) as connection:
                        return connection.getpeercert()
            try:
                cert = handshake(ssl.create_default_context(cafile=str(certificate)), '43.135.142.53')
            except ssl.SSLError:
                raise RuntimeError('public-ip-tls-unavailable') from None
            if ('IP Address', '43.135.142.53') not in cert.get('subjectAltName', ()):
                raise RuntimeError('public-ip-san-missing')
            for context, hostname in [(ssl.create_default_context(), '43.135.142.53'),
                                      (ssl.create_default_context(cafile=str(certificate)), '43.135.142.54')]:
                try:handshake(context, hostname)
                except ssl.SSLCertVerificationError:pass
                else:raise RuntimeError('invalid-ip-certificate-accepted')
            def upstream_ready():
                try:
                    docker('exec', upstream, 'node', '-e',
                           'fetch("http://127.0.0.1:8080/readyz").then(r=>{if(r.status!==200)process.exit(1)}).catch(()=>process.exit(1))')
                    return True
                except RuntimeError:return False
            for _ in range(20):
                if upstream_ready():break
                time.sleep(.25)
            else:raise RuntimeError('route-upstream-not-ready')
            for method, path, source in [('GET', '/readyz', 'api'), ('GET', '/healthz', 'api'),
                                         ('GET', '/login', 'web'), ('GET', '/readyz/private', 'web'),
                                         ('GET', '/api/v1/content/private', 'web'), ('POST', '/readyz', 'web')]:
                with socket.create_connection(('127.0.0.1', port), timeout=5) as transport:
                    with ssl.create_default_context(cafile=str(certificate)).wrap_socket(
                            transport, server_hostname='43.135.142.53') as connection:
                        connection.sendall((method+' '+path+' HTTP/1.0\r\nHost: 43.135.142.53\r\nContent-Length: 0\r\nConnection: close\r\n\r\n').encode())
                        raw = b''
                        while True:
                            chunk = connection.recv(8192)
                            if not chunk:break
                            raw += chunk
                            if len(raw)>16384:raise RuntimeError('route-response-too-large')
                headers, separator, body = raw.partition(b'\r\n\r\n')
                if not separator or b' 200 ' not in headers.split(b'\r\n', 1)[0] or body != (source+':'+path).encode():
                    diagnostic=subprocess.run(['docker','logs','--tail','4',name],capture_output=True,timeout=10)
                    print(json.dumps({'fixtureRoute':{'method':method,'path':path,'status':headers.split(b'\r\n',1)[0].decode(errors='replace'),'body':body.decode(errors='replace')[:200],'gatewayLog':(diagnostic.stdout+diagnostic.stderr).decode(errors='replace')[-800:]}}),flush=True)
                    raise RuntimeError('gateway-health-route-invalid')
            return {'ipClientTLS': True, 'unknownCARejected': True, 'wrongIPRejected': True,
                    'tlsFixtureCertificateOnly': True, 'exactHealthRoutes': True,
                    'applicationRoutesPreserved': True}
        finally:
            if created:docker('rm', '--force', name)
            if upstream_created:docker('rm', '--force', upstream)
            if network_created:docker('network', 'rm', network)


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
           'test "$(id -u)" -ne 0; for name in server migrate admin-init correction-maintenance topic-catalogue-import topic-learning-maintenance; do test -x /app/bin/$name; done; test ! -e /app/bin/e2e-harness; test -f /app/db/migrations/00008_correction_notifications.sql; test -f /app/db/migrations/00012_topic_cutover.sql')
    version = docker('run', '--rm', '--network', 'none', '--read-only', images['gateway'], 'version').decode().strip()
    if not version.startswith('v2.11.7 '):
        raise RuntimeError('gateway-version-mismatch')
    tls_result = check_ip_tls(images['gateway'], images['web'])
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
        return {'ok': True, 'revision': revision, 'apiWhitelist': True, 'harnessAbsent': True, 'gatewayVersion': version.split()[0], **tls_result, **result}
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
