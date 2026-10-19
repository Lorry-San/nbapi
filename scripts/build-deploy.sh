#!/usr/bin/env bash
# Build an uploaded source release, then install it and verify after a 30-second wait.
set -Eeuo pipefail
version=${1:?Usage: build-deploy.sh VERSION RELEASE_DIRECTORY}
release=${2:?Release directory is required}
[[ "$version" =~ ^[0-9A-Za-z._-]+$ ]]
[[ "$release" == /opt/lightcone/builds/* && -d "$release/web" ]]
cd "$release"
export PATH="/opt/lightcone/toolchains/node/bin:/opt/lightcone/toolchains/bun:/opt/lightcone/toolchains/go/bin:$PATH"
export GOMODCACHE=/opt/lightcone/build-cache/gomod
export GOCACHE=/opt/lightcone/build-cache/gocache
export BUN_INSTALL_CACHE_DIR=/opt/lightcone/build-cache/bun
export GOMAXPROCS=2 GOFLAGS=-p=2 CGO_ENABLED=0
export NODE_OPTIONS=--max-old-space-size=4096
export RAYON_NUM_THREADS=2
printf 'Building web frontend for %s\n' "$version"
(cd web && bun install --frozen-lockfile && bun run typecheck && bun test src/features/lightcode-usage && bun run build)
printf 'Testing gateway routes and activity data\n'
go test ./model ./router ./controller
printf 'Compiling gateway %s\n' "$version"
go build -trimpath -ldflags "-s -w -X github.com/Lorry-San/nbapi/common.Version=$version" -o "$release/lightcone-ai" .
test -s "$release/lightcone-ai"
grep -qx 'LIGHTCODE_ENFORCE_CLIENT=0' /etc/lightcone.env
stamp=$(date +%Y%m%d%H%M%S)
backup="/opt/lightcone/bin/lightcone-ai.before-$version-$stamp"
cp -a /opt/lightcone/bin/lightcone-ai "$backup"
rollback() {
  result=$?
  trap - ERR
  echo "Update failed; restoring previous executable"
  systemctl stop lightcone || true
  install -o root -g root -m 755 "$backup" /opt/lightcone/bin/lightcone-ai
  systemctl start lightcone
  exit "$result"
}
trap rollback ERR
systemctl stop lightcone
# SQLite's backup API includes WAL state if the deployment enables it.
DB_BACKUP="/opt/lightcone/data/lightcone.db.bak-$stamp" python3 - <<'PY'
import os, sqlite3
with sqlite3.connect('/opt/lightcone/data/lightcone.db') as source:
    with sqlite3.connect(os.environ['DB_BACKUP']) as target:
        source.backup(target)
PY
install -o root -g root -m 755 "$release/lightcone-ai" /opt/lightcone/bin/lightcone-ai
systemctl start lightcone
printf 'Updated service; waiting 30 seconds before verification...\n'
sleep 30
systemctl is-active --quiet lightcone
curl -fsS http://127.0.0.1:3000/api/status | EXPECTED_VERSION="$version" python3 -c 'import json,os,sys; data=json.load(sys.stdin); actual=data["data"]["version"]; assert actual==os.environ["EXPECTED_VERSION"],actual; print("Healthy gateway:",actual)'
status=$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:3000/lightcode)
[[ "$status" == 308 ]]
trap - ERR
sha256sum /opt/lightcone/bin/lightcone-ai
printf 'Deployment verified. Executable backup: %s\nDatabase backup: /opt/lightcone/data/lightcone.db.bak-%s\n' "$backup" "$stamp"
