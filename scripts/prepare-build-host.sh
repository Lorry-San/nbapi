#!/usr/bin/env bash
# Install checksum-pinned build tools into the application-owned toolchain directory.
set -Eeuo pipefail
mkdir -p /opt/lightcone/toolchains /opt/lightcone/builds /opt/lightcone/build-cache
python3 - <<'PY'
import hashlib, io, pathlib, tarfile, urllib.request, zipfile
root = pathlib.Path('/opt/lightcone/toolchains')
tools = [
    ('go', 'https://go.dev/dl/go1.27.3.linux-amd64.tar.gz', '65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40', 'go/bin/go'),
    ('node', 'https://nodejs.org/dist/v24.10.0/node-v24.10.0-linux-x64.tar.xz', '2642f4428869aca32443660fd71b3918e2be1277a899bdcaeb64c93b54b5af17', 'node/bin/node'),
    ('bun', 'https://github.com/oven-sh/bun/releases/download/bun-v1.4.2/bun-linux-x64.zip', '36368faef7527875d5ffa52e53cd48021741f2a83eb6208a8dd64068d422a913', 'bun/bun'),
]
for name, url, checksum, executable in tools:
    marker = root / (name + '.sha256')
    if (root / executable).is_file() and marker.exists() and marker.read_text() == checksum:
        print(name, 'toolchain already verified', flush=True)
        continue
    print('Downloading', name, flush=True)
    data = urllib.request.urlopen(url, timeout=180).read()
    if hashlib.sha256(data).hexdigest() != checksum:
        raise RuntimeError(name + ' archive checksum mismatch')
    # An upgrade extracts over the previous tree, so files dropped between releases survive and
    # break the build (Go 1.27 adds map_swiss.go while the old map.go still declares ctrlEmpty).
    # Replace the tool's own directory instead of merging into it.
    import shutil
    target = root / name
    if target.exists():
        shutil.rmtree(target)
    marker.unlink(missing_ok=True)
    if name == 'bun':
        with zipfile.ZipFile(io.BytesIO(data)) as archive:
            target = root / executable
            target.parent.mkdir(exist_ok=True)
            target.write_bytes(archive.read('bun-linux-x64/bun'))
            target.chmod(0o755)
    else:
        with tarfile.open(fileobj=io.BytesIO(data), mode='r:*') as archive:
            members = archive.getmembers()
            prefix = 'node-v24.10.0-linux-x64' if name == 'node' else 'go'
            for member in members:
                parts = pathlib.PurePosixPath(member.name).parts
                if not parts or parts[0] != prefix or '..' in parts:
                    raise RuntimeError('Unexpected archive member: ' + member.name)
                member.name = str(pathlib.PurePosixPath(name, *parts[1:]))
                if member.issym() or member.islnk():
                    target = (root / member.name).parent / member.linkname
                    if member.islnk():
                        target = root / member.linkname
                    if root.resolve() not in target.resolve().parents:
                        raise RuntimeError('Archive link outside toolchains: ' + member.name)
            # These checksum-pinned archives are also supported on Python 3.8.
            archive.extractall(root, members=members)
    marker.write_text(checksum)
PY
/opt/lightcone/toolchains/go/bin/go version
/opt/lightcone/toolchains/node/bin/node --version
/opt/lightcone/toolchains/bun/bun --version
