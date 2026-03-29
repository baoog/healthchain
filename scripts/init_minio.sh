#!/usr/bin/env bash
set -euo pipefail
CID=$(docker ps -qf "name=minio")
[ -z "$CID" ] && { echo "MinIO not running"; exit 1; }

docker exec "$CID" sh -lc '
apk add --no-cache mc >/dev/null
mc alias set local http://localhost:9000 admin adminadmin
mc mb -p local/ehr || true
mc version enable local/ehr
mc ilm add --id retain-7y --expiry-days 2555 local/ehr || true
echo "✅ MinIO bucket ready"
'