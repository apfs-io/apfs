# Service Initialization

APFS performs several initialization steps automatically when the `server` or
`processor` command starts. No external seed scripts are required for a typical
Docker deployment.

---

## Startup sequence

When you run `apfs server` (optionally with `--processing=1` to embed a worker),
the service initializes in this order:

```
1. Load configuration (env, CLI flags, optional config file)
2. Connect meta database          STORAGE_METADB_CONNECT
3. Connect file storage driver    STORAGE_CONNECT  (fs:// or s3://)
4. Connect processing state store STORAGE_STATE_CONNECT
5. Load procedure store           STORAGE_PROCEDURE_DIR
6. Register workflow step runners STORAGE_CONVERTERS + WORKER_TAGS
7. Bootstrap bucket workflows     WORKFLOWS_DIR
8. Start gRPC / HTTP listeners    SERVER_GRPC_LISTEN, SERVER_HTTP_LISTEN
9. (optional) Subscribe to event stream and run embedded processor
```

Steps 2–7 happen inside `api.NewServer` before the API accepts traffic.
If workflow bootstrap fails (parse error, storage write error), the process
exits and the container will not become healthy.

The standalone `apfs processor` command runs the same storage and workflow
bootstrap path because it shares the same `ProtocolAPIObject` initialization.

---

## Configuration sources

Configuration is loaded via [goconfig](https://github.com/demdxx/goconfig):

| Source                    | Priority |
| ------------------------- | -------- |
| CLI flags                 | highest  |
| Environment variables     | high     |
| Config file (`-config=…`) | medium   |
| Struct defaults           | lowest   |

Run with `LOG_LEVEL=debug` to print the resolved configuration on startup.

---

## Workflow bootstrap

Bucket-level workflows (manifests) can be applied automatically from a
filesystem directory. This replaces the legacy `deploy/init/seed-workflows.sh`
sidecar pattern.

### Directory layout

```
{WORKFLOWS_DIR}/
  {groupName}/
    manifest.yaml    ← preferred
    manifest.yml
    manifest.json
```

- **`WORKFLOWS_DIR`** — root directory (default: `/workflows`).
- **`{groupName}`** — APFS group / bucket name (`images`, `videos`, …).
- The manifest filename must be exactly one of the three names above.

Example from this repository:

```
deploy/workflows/
  images/manifest.yaml
  analysis/manifest.yaml
  avatars/manifest.yaml
  videos/manifest.yaml
```

### Apply rules

For each subdirectory that contains a manifest, APFS reads the current workflow
from storage and decides:

| Stored workflow        | `WORKFLOWS_RECONFIGURE` | Incoming manifest             | Result    |
| ---------------------- | ----------------------- | ----------------------------- | --------- |
| not configured (empty) | any                     | valid                         | **Apply** |
| configured             | `false` (default)       | any                           | **Skip**  |
| configured             | `true`                  | `version` greater than stored | **Apply** |
| configured             | `true`                  | same or lower `version`       | **Skip**  |

A group is **not configured** when storage returns an empty workflow (no jobs
and no `validate` block).

The `version` field in the manifest is a user-defined deployment version
(dotted numeric strings are compared: `2` < `2.1` < `3`). It is separate from
the v2 schema identifier (`version: "2"` in examples denotes the YAML schema,
but the same field drives upgrade detection when you bump it).

### Environment variables

| Variable                | Default      | Description                                                         |
| ----------------------- | ------------ | ------------------------------------------------------------------- |
| `WORKFLOWS_DIR`         | `/workflows` | Directory with per-group manifests. Empty value disables bootstrap. |
| `WORKFLOWS_RECONFIGURE` | `false`      | Allow upgrading existing groups when incoming version is newer.     |

### Behaviour edge cases

- **Directory missing** — bootstrap is skipped silently (debug log). Useful when
  `/workflows` is not mounted and you configure workflows via API instead.
- **Subdirectory without manifest** — skipped (debug log).
- **Hidden directories** (name starts with `.`) — ignored.
- **Empty manifest** (no jobs, no validate) — skipped (warning log).
- **Parse error** — startup fails with an error.

### Storage mapping

After bootstrap, manifests are stored where the active driver expects them:

| Driver  | Stored as                        |
| ------- | -------------------------------- |
| `fs://` | `{root}/{group}/manifest.yaml`   |
| `s3://` | `{bucket}/{group}/manifest.json` |

Bootstrap uses the same `SetWorkflow` path as the HTTP `PUT /v1/manifest/{group}`
API.

---

## Procedure store

Named `procedure` steps load `.eproc.yaml` manifests from disk at startup.

| Variable                | Default      | Description                                            |
| ----------------------- | ------------ | ------------------------------------------------------ |
| `STORAGE_PROCEDURE_DIR` | `procedures` | Directory scanned for `*.eproc.yaml` / `*.eproc.json`. |

If the directory is empty or missing, procedure/shell/exec/docker steps are
disabled and a warning is logged. Production images copy `deploy/procedures` to
`/procedures`.

---

## Converters and worker tags

| Variable             | Default           | Description                                                            |
| -------------------- | ----------------- | ---------------------------------------------------------------------- |
| `STORAGE_CONVERTERS` | `image,procedure` | Comma-separated list: `image`, `procedure`, `shell`, `exec`, `docker`. |
| `WORKER_TAGS`        | _(empty)_         | Worker capability tags matched against job `runs-on:` values.          |

Both are resolved during step-runner registration, before workflow bootstrap.

---

## Docker deployment

### Mount workflows at runtime

```yaml
services:
  apfs:
    image: github.com/apfs-io/apfs:ubuntu-imagemagick
    environment:
      WORKFLOWS_DIR: /workflows
      WORKFLOWS_RECONFIGURE: "false"
      STORAGE_CONNECT: s3://minio:9000/assets?access=…&secret=…&insecure=true
    volumes:
      - ./my-workflows:/workflows:ro
```

Edit files under `./my-workflows/{group}/manifest.yaml` and restart the
container. With `WORKFLOWS_RECONFIGURE=false`, already-configured groups are
left unchanged; only new groups are seeded.

To roll out manifest updates, either:

1. Set `WORKFLOWS_RECONFIGURE=true` and bump `version` in the manifest, or
2. Update workflows via `PUT /v1/manifest/{group}` without restarting.

### Bake workflows into the image

Production Dockerfiles include:

```dockerfile
ENV WORKFLOWS_DIR=/workflows
COPY deploy/workflows /workflows
```

No volume mount is required; manifests are applied on first start.

### Local development stack

`deploy/develop/docker-compose.yml` mounts `deploy/workflows` at `/workflows`
and starts APFS without a separate seed container. See
[deploy/README.md](../deploy/README.md).

---

## Manual alternatives

Use these when bootstrap is disabled (`WORKFLOWS_DIR=` empty) or for one-off
updates:

| Method            | Tool                                                                |
| ----------------- | ------------------------------------------------------------------- |
| HTTP API          | `PUT /v1/manifest/{group}` — see [WORKFLOW.md](WORKFLOW.md)         |
| Helper script     | `deploy/init/upload_workflow.py`                                    |
| Legacy S3 upload  | `deploy/init/seed-workflows.sh` (deprecated; same directory layout) |
| Filesystem driver | Copy `manifest.yaml` into `{storage_root}/{group}/`                 |

---

## Logging

Bootstrap emits structured log lines (requires `LOG_LEVEL=info` or lower):

```
workflows bootstrap: applied workflow  group=images version=2 reason="group not configured"
workflows bootstrap: skip group        group=images reason="group already configured and reconfigure disabled"
workflows bootstrap: applied workflow  group=images version=3 reason="incoming version is newer"
```

Set `LOG_LEVEL=debug` to see skipped directories and groups without manifests.

---

## Related documentation

| Document                                | Content                                         |
| --------------------------------------- | ----------------------------------------------- |
| [WORKFLOW.md](WORKFLOW.md)              | Workflow YAML schema                            |
| [deploy/README.md](../deploy/README.md) | Docker images, compose stack, example manifests |
| [USE_CASES.md](USE_CASES.md)            | End-to-end workflow examples                    |
