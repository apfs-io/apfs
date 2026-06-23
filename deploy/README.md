# Deploy

Deployment assets for APFS: Docker images, local development stack, workflow
examples, and external procedure manifests.

## Layout

```
deploy/
├── build.mk                 # Cross-platform build helpers (included from Makefile)
├── develop/                 # Local docker-compose stack
│   ├── apfs.dockerfile      # Dev server image (ImageMagick + Docker CLI)
│   ├── docker-compose.yml   # MinIO, Redis, NATS, APFS server, seed, testapp
│   └── testapp.dockerfile
├── init/
│   └── seed-workflows.sh    # Upload workflow YAML → S3 manifest.json (dev stack)
├── procedures/              # plugeproc .eproc manifests + scripts
│   ├── *.eproc.yaml
│   ├── image-resize-w
│   ├── image-strip-meta
│   ├── image2vec.py
│   ├── facemeta.py
│   └── requirements.txt     # Python deps for ML procedures
├── production/              # Production Docker images
│   ├── scratch.dockerfile
│   ├── debian.dockerfile
│   ├── ubuntu.dockerfile
│   └── ubuntu-imagemagick.dockerfile
└── workflows/               # v2 workflow examples (YAML)
    ├── image-gallery.yaml
    ├── image-analysis.yaml
    ├── user-avatar.yaml
    └── video-transcode.yaml
```

## Workflow examples

Each file in `workflows/` is a **v2 workflow** (`version: "2"`). Upload it to a
bucket/group as `manifest.yaml` (filesystem driver) or `manifest.json` (S3 driver).

| File                  | APFS group | Purpose                                      |
| --------------------- | ---------- | -------------------------------------------- |
| `image-gallery.yaml`  | `images`   | procedure + shell + docker resize pipeline   |
| `image-analysis.yaml` | `analysis` | ML embedding + face detection + dimensions   |
| `user-avatar.yaml`    | `avatars`  | Avatar + micro thumbnail                     |
| `video-transcode.yaml`| `videos`   | FFmpeg Docker transcode + thumbnail          |

See [docs/WORKFLOW.md](../docs/WORKFLOW.md) for the full schema.

### Step execution modes

Workflow steps use the `proc` runner ([libs/converters/proc](../libs/converters/proc/)):

| `uses:`     | Description                                              |
| ----------- | -------------------------------------------------------- |
| `procedure` | Named `.eproc.yaml` from `deploy/procedures/`            |
| `shell`     | Inline bash via `run:`                                   |
| `exec`      | Alias for `procedure`                                    |
| `docker`    | Command in container (`docker:` block + optional `run:`) |
| `image/*`   | Built-in image converter (resize, blur, …)               |

### Worker tags (`runs-on`)

Jobs declare affinity via `runs-on:`. The worker handles a job when one of its
`WORKER_TAGS` matches:

```env
WORKER_TAGS=image,gpu,cpu,docker,video,any
```

| Tag      | Typical worker setup                          |
| -------- | --------------------------------------------- |
| `image`  | ImageMagick installed (resize, strip EXIF)    |
| `gpu`    | Python + torch (image2vec)                    |
| `cpu`    | Python + OpenCV (facemeta)                    |
| `docker` | Docker daemon available                       |
| `video`  | Docker + FFmpeg image pulled on demand        |
| `any`    | Matches all jobs with `runs-on: any`          |

## Procedures

Procedures are loaded from `STORAGE_PROCEDURE_DIR` (default `/procedures`).
Each procedure has a companion `.eproc.yaml` manifest consumed by
[plugeproc](https://github.com/demdxx/plugeproc).

Example workflow step:

```yaml
steps:
  - name: Resize
    uses: procedure
    with:
      name: image-resize-w
      width: "1200"
      target: large.jpg
```

Parameters in `with:` map to positional args declared in the `.eproc.yaml`
manifest. Reserved keys: `target`, `target-meta`, `input`, `tojson`, `name`.

## Local development

```bash
# Build and start the stack (MinIO + APFS + workflow seed + testapp)
make build-docker-dev
make run

# Upload a test image manually
make test-upload

# Put workflow for the images group via HTTP (legacy manifest endpoint)
make test-workflow

# Read it back
make test-get-workflow
```

The `seed` service in docker-compose uploads all workflow examples to MinIO as
`{group}/manifest.json` before the testapp starts.

### Required environment

Key variables (see `.env`):

```env
STORAGE_PROCEDURE_DIR=/procedures/
STORAGE_CONVERTERS=image,procedure,shell,exec,docker
WORKER_TAGS=image,gpu,cpu,docker,video,any
STORAGE_CONNECT=s3://s3server:9000/assets?...
```

For Docker workflow steps, mount the host socket in compose:

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock
```

## Production images

| Image                    | Base              | Includes                          |
| ------------------------ | ----------------- | --------------------------------- |
| `scratch.dockerfile`     | scratch           | Binary + procedures only          |
| `debian.dockerfile`      | debian:stable-slim| Binary + procedures               |
| `ubuntu.dockerfile`      | ubuntu:plucky     | Binary + procedures               |
| `ubuntu-imagemagick`     | ubuntu:plucky     | + ImageMagick, Python ML deps     |

Build all production variants:

```bash
make buildx-docker-production
```

Pick worker tags per deployment — e.g. a GPU node runs with
`WORKER_TAGS=gpu,any`, an image node with `WORKER_TAGS=image,any`.

## Uploading a workflow manually

### Filesystem driver

Copy YAML to the bucket root:

```bash
cp deploy/workflows/image-gallery.yaml /data/storage/images/manifest.yaml
```

### HTTP API

```bash
# Convert YAML to JSON for S3-backed storage
curl -X PUT -H 'Content-Type: application/json' \
  --data-binary "@<(yq -o=json deploy/workflows/image-gallery.yaml)" \
  "http://localhost:18080/v1/manifest/images"
```

The server stores the workflow internally regardless of the legacy manifest wire format.
