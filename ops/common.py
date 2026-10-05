"""部署配置与 Docker 边界；从不执行环境文件或打印命令输出。"""
from pathlib import Path
import os
import re
import stat
import subprocess

PROJECT = 'math-master-preview'
ORIGIN = 'https://43.135.142.53'
ACME = {
    'staging': 'https://acme-staging-v02.api.letsencrypt.org/directory',
    'production': 'https://acme-v02.api.letsencrypt.org/directory',
}
KEYS = {'AUTH_PUBLIC_ORIGIN', 'POSTGRES_DB', 'POSTGRES_USER', 'POSTGRES_PASSWORD', 'DB_LC_COLLATE', 'DB_LC_CTYPE'}


class ConfigError(ValueError):
    pass


def safe_path(path: Path) -> Path:
    absolute = path.absolute()
    if '..' in path.parts:
        raise ConfigError('unsafe-path')
    for candidate in [absolute, *absolute.parents]:
        if candidate.is_symlink():
            raise ConfigError('unsafe-path')
    return absolute


def validate_revision(revision: str) -> str:
    if not re.fullmatch(r'[0-9a-f]{40}', revision):
        raise ConfigError('invalid-release')
    return revision


def load_env(path: Path) -> dict[str, str]:
    path = safe_path(path)
    metadata = path.lstat()
    if not stat.S_ISREG(metadata.st_mode) or stat.S_IMODE(metadata.st_mode) != 0o600 or metadata.st_uid != os.geteuid():
        raise ConfigError('unsafe-environment-file')
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, 'r', encoding='utf-8') as stream:
        text = stream.read(8193)
    if len(text) > 8192:
        raise ConfigError('invalid-environment')
    values: dict[str, str] = {}
    for line in text.splitlines():
        if not line.strip() or line.lstrip().startswith('#'):
            continue
        key, separator, value = line.partition('=')
        if not separator or key not in KEYS or key in values or not value or any(c in value for c in '$`\\\"\''):
            raise ConfigError('invalid-environment')
        values[key] = value
    if set(values) != KEYS or values['AUTH_PUBLIC_ORIGIN'] != ORIGIN:
        raise ConfigError('invalid-production-origin')
    if any(values[key] != 'math_master_preview' for key in ['POSTGRES_DB', 'POSTGRES_USER']):
        raise ConfigError('invalid-database-identity')
    if not re.fullmatch(r'[0-9a-f]{64}', values['POSTGRES_PASSWORD']):
        raise ConfigError('invalid-database-secret')
    for key in ['DB_LC_COLLATE', 'DB_LC_CTYPE']:
        if not re.fullmatch(r'[A-Za-z0-9_.@-]{1,64}', values[key]):
            raise ConfigError('invalid-database-locale')
    return values


def clean_environment() -> dict[str, str]:
    # 保留本机 Docker 客户端查找配置所需的既有 HOME/PATH，不继承部署或远程 daemon 覆盖。
    return {key: os.environ[key] for key in ['PATH', 'HOME', 'LANG', 'LC_ALL'] if key in os.environ}


def docker_command(args: list[str], sudo: bool = False) -> list[str]:
    return (['sudo', '-n', 'docker'] if sudo else ['docker']) + args


def docker(args: list[str], sudo: bool = False, *, input: bytes | None = None) -> subprocess.CompletedProcess[bytes]:
    return subprocess.run(docker_command(args, sudo), input=input, capture_output=True, env=clean_environment(), timeout=120)


def compose(root: Path, revision: str, args: list[str], sudo: bool = False, acme: str = 'production') -> subprocess.CompletedProcess[bytes]:
    root = safe_path(root)
    validate_revision(revision)
    if acme not in ACME:
        raise ConfigError('invalid-acme-environment')
    release = safe_path(root / 'releases' / revision)
    config = safe_path(root / 'shared' / 'configs' / (revision + '.env'))
    load_env(config)
    compose_file = safe_path(release / 'compose.yaml')
    if not compose_file.is_file():
        raise ConfigError('missing-release')
    dynamic = {
        'RELEASE_SHA': revision, 'PUBLIC_HOST': '43.135.142.53',
        'ACME_DIRECTORY': ACME[acme], 'CADDY_DATA_VOLUME': f'{PROJECT}-caddy-{acme}',
    }
    command = ['docker', 'compose', '--project-name', PROJECT, '--project-directory', str(release), '--env-file', str(config), '-f', str(compose_file), *args]
    environment = clean_environment()
    if sudo:
        command = ['sudo', '-n', 'env', *[f'{k}={v}' for k, v in dynamic.items()], *command]
    else:
        environment.update(dynamic)
    return subprocess.run(command, capture_output=True, env=environment, timeout=530)
