#!/bin/sh
# Seed bucket-level workflow manifests into MinIO (S3 backend).
# See deploy/README.md for group mapping.

set -eu

if command -v python3 >/dev/null 2>&1; then
  PY=python3
else
  PY=python
fi

# Install deps when running in minimal containers (Alpine seed service).
if ! "$PY" -c "import yaml, boto3" 2>/dev/null; then
  if command -v apk >/dev/null 2>&1; then
    apk add --no-cache py3-pip >/dev/null
    pip install --quiet pyyaml boto3
  else
    echo "install pyyaml and boto3 for Python" >&2
    exit 1
  fi
fi

export S3_ENDPOINT="${S3_ENDPOINT:-http://s3server:9000}"
export S3_BUCKET="${S3_BUCKET:-assets}"
export WORKFLOWS_DIR="${WORKFLOWS_DIR:-/workflows}"

"$PY" <<'PY'
import json
import os
import sys
import time
from pathlib import Path

import boto3
import yaml
from botocore.client import Config

endpoint = os.environ.get("S3_ENDPOINT", "http://s3server:9000")
bucket = os.environ.get("S3_BUCKET", "assets")
access = os.environ["S3_ACCESS_KEY"]
secret = os.environ["S3_SECRET_KEY"]
workflows_dir = Path(os.environ.get("WORKFLOWS_DIR", "/workflows"))

mapping = {
    "image-gallery.yaml": "images",
    "image-analysis.yaml": "analysis",
    "user-avatar.yaml": "avatars",
    "video-transcode.yaml": "videos",
}

client = boto3.client(
    "s3",
    endpoint_url=endpoint,
    aws_access_key_id=access,
    aws_secret_access_key=secret,
    region_name="default",
    config=Config(signature_version="s3v4"),
)

for _ in range(30):
    try:
        client.list_buckets()
        break
    except Exception:
        time.sleep(1)
else:
    sys.exit("MinIO not ready")

try:
    client.head_bucket(Bucket=bucket)
except client.exceptions.ClientError:
    client.create_bucket(Bucket=bucket)

for filename, group in mapping.items():
    path = workflows_dir / filename
    if not path.exists():
        print(f"skip missing {path}")
        continue
    with path.open() as f:
        doc = yaml.safe_load(f)
    key = f"{group}/manifest.json"
    body = json.dumps(doc, ensure_ascii=False).encode()
    client.put_object(
        Bucket=bucket,
        Key=key,
        Body=body,
        ContentType="application/json",
    )
    print(f"uploaded s3://{bucket}/{key} from {filename}")
PY
