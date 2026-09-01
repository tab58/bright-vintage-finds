#!/bin/sh
# Re-register orphaned S3 objects with floci.
#
# Floci hybrid mode persists writes to ./data/s3/<bucket>/<key> but does
# not rescan the directory tree on boot — only objects PUT through the API
# this session appear in its index. When bytes on disk outlive the floci
# container's in-memory state (fresh container, storage mode change, etc.),
# HEAD/GET return 404 even though the file is right there.
#
# This script walks every on-disk bucket under ./data/s3/ and re-PUTs
# each object through the S3 API so floci registers them. The bytes on
# disk are unchanged (same key, same content).
#
# When run as a Floci init hook (mounted at /etc/floci/init/start.d/),
# the S3 data is at /app/data. When run from the host, it's relative to
# the script location.
#
# Usage: ./reregister-floci-orphans.sh
# Requires: floci running on localhost:4566, awscli installed.

if [ -d /app/data/s3 ]; then
  S3_ROOT="/app/data/s3"
else
  case "$0" in
    /*) SCRIPT_DIR="$(dirname "$0")" ;;
    *)  SCRIPT_DIR="$(pwd)/$(dirname "$0")" ;;
  esac
  S3_ROOT="$SCRIPT_DIR/s3"
fi

ENDPOINT="http://localhost:4566"

export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1

if [ ! -d "$S3_ROOT" ]; then
  echo "No on-disk floci data at $S3_ROOT — nothing to do."
  exit 0
fi

if ! aws --endpoint-url="$ENDPOINT" s3api list-buckets >/dev/null 2>&1; then
  echo "floci not reachable at $ENDPOINT — start it with: task env-up"
  exit 1
fi

for bucket_dir in "$S3_ROOT"/*; do
  [ -d "$bucket_dir" ] || continue
  bucket="$(basename "$bucket_dir")"
  [ "$bucket" = ".bucketstack" ] && continue

  echo "== bucket: $bucket"
  aws --endpoint-url="$ENDPOINT" s3api create-bucket --bucket "$bucket" >/dev/null 2>&1 || true

  find "$bucket_dir" -type f ! -path "*/.bucketstack/*" ! -name "*.s3data" | while IFS= read -r file; do
    key="${file#$bucket_dir/}"
    echo "  + $key"
    aws --endpoint-url="$ENDPOINT" s3 cp "$file" "s3://$bucket/$key" >/dev/null 2>&1 || true
  done
done

echo "done."
