"""Upload tracked source, build on the server, update the service, wait 30s and verify.

Run: python scripts/deploy-lightcone.py --host HOST --version VERSION
Authentication uses SSH keys, LC_SSH_PASSWORD, or an interactive password prompt.
"""
import argparse
import getpass
import hashlib
import os
from pathlib import Path
import re
import shlex
import subprocess
import tarfile
import tempfile
import time

import paramiko

ROOT = Path(__file__).resolve().parents[1]


def connect(host: str, user: str) -> paramiko.SSHClient:
    client = paramiko.SSHClient()
    client.load_system_host_keys()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    password = os.environ.get('LC_SSH_PASSWORD')
    if not password and not os.environ.get('SSH_AUTH_SOCK'):
        password = getpass.getpass(f'SSH password for {user}@{host} (empty for keys): ')
    client.connect(host, username=user, password=password or None, timeout=30)
    return client


def execute(client: paramiko.SSHClient, command: str) -> None:
    stdin, stdout, stderr = client.exec_command(command + ' 2>&1', timeout=None)
    stdin.close()
    try:
        for line in stdout:
            print(line, end='', flush=True)
        result = stdout.channel.recv_exit_status()
        if result:
            raise RuntimeError(f'Remote command exited {result}')
    finally:
        stdout.close()
        stderr.close()


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--host', default=os.environ.get('LC_SSH_HOST'))
    parser.add_argument('--user', default='root')
    parser.add_argument('--version', required=True)
    parser.add_argument('--prepare-only', action='store_true')
    args = parser.parse_args()
    if not args.host or not re.fullmatch(r'[0-9A-Za-z._-]+', args.version):
        parser.error('A host and a safe release version are required')
    client = connect(args.host, args.user)
    try:
        with client.open_sftp() as sftp:
            sftp.put(str(ROOT / 'scripts/prepare-build-host.sh'), '/root/lightcone-prepare-build.sh')
        execute(client, 'bash /root/lightcone-prepare-build.sh')
        if args.prepare_only:
            return
        tracked = subprocess.check_output(['git', 'ls-files', '-z'], cwd=ROOT).decode().split('\0')
        files = set(filter(None, tracked)) | {
            'scripts/build-deploy.sh', 'scripts/prepare-build-host.sh', 'scripts/deploy-lightcone.py',
        }
        with tempfile.TemporaryDirectory(prefix='lightcone-deploy-') as directory:
            bundle = Path(directory) / 'source.tar.gz'
            with tarfile.open(bundle, 'w:gz') as archive:
                for name in sorted(files):
                    source = ROOT / name
                    if source.is_file():
                        archive.add(source, arcname=name, recursive=False)
            checksum = hashlib.sha256(bundle.read_bytes()).hexdigest()
            release = f'/opt/lightcone/builds/{args.version}-{time.strftime("%Y%m%d%H%M%S")}'
            remote_bundle = release + '.tar.gz'
            execute(client, 'mkdir -p ' + shlex.quote(release))
            with client.open_sftp() as sftp:
                sftp.put(str(bundle), remote_bundle)
            verify = f"printf '%s  %s\\n' {shlex.quote(checksum)} {shlex.quote(remote_bundle)} | sha256sum -c -"
            unpack = f'tar -xzf {shlex.quote(remote_bundle)} -C {shlex.quote(release)}'
            build = f'bash {shlex.quote(release + "/scripts/build-deploy.sh")} {shlex.quote(args.version)} {shlex.quote(release)}'
            execute(client, f'{verify} && {unpack} && {build}')
    finally:
        client.close()


if __name__ == '__main__':
    main()
