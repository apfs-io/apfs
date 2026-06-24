#!/usr/bin/env python3
"""Upload a v2 workflow YAML file to object storage as manifest.json.

Usage:
  upload_workflow.py <group> <workflow.yaml>

Environment (S3 / MinIO):
  S3_ENDPOINT   default http://localhost:9000
  S3_BUCKET       default assets
  S3_ACCESS_KEY   required
  S3_SECRET_KEY   required

The workflow is stored at: s3://{bucket}/{group}/manifest.json
"""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("pip install pyyaml", file=sys.stderr)
    raise

try:
    import boto3
    from botocore.client import Config
except ImportError:
    print("pip install boto3", file=sys.stderr)
    raise


def main() -> None:
    if len(sys.argv) != 3:
        print(f"usage: {sys.argv[0]} <group> <workflow.yaml>", file=sys.stderr)
        sys.exit(2)

    group = sys.argv[1]
    path = Path(sys.argv[2])
    if not path.is_file():
        print(f"file not found: {path}", file=sys.stderr)
        sys.exit(1)

    endpoint = os.environ.get("S3_ENDPOINT", "http://localhost:9000")
    bucket = os.environ.get("S3_BUCKET", "assets")
    access = os.environ["S3_ACCESS_KEY"]
    secret = os.environ["S3_SECRET_KEY"]

    with path.open() as f:
        doc = yaml.safe_load(f)

    client = boto3.client(
        "s3",
        endpoint_url=endpoint,
        aws_access_key_id=access,
        aws_secret_access_key=secret,
        region_name="default",
        config=Config(signature_version="s3v4"),
    )

    key = f"{group}/manifest.json"
    body = json.dumps(doc, ensure_ascii=False).encode()
    client.put_object(
        Bucket=bucket,
        Key=key,
        Body=body,
        ContentType="application/json",
    )
    print(f"uploaded s3://{bucket}/{key} from {path}")


if __name__ == "__main__":
    main()
