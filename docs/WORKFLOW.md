# Workflow Reference

A **workflow** is a YAML file (`manifest.yaml`) stored at the root of a bucket that declares how uploaded objects are validated and processed. It replaces the legacy v1 `manifest.json`.

The schema is inspired by GitHub Actions: jobs form a DAG via `needs:`, support `if:` conditions, and can be assigned to specific worker pools via `runs-on:`.

---

## Top-level keys

```yaml
version: "2" # required; identifies the v2 schema
name: "Image pipeline" # optional human-readable label
description: | # optional multi-line description
  Resize uploaded images to three sizes and generate a blurred preview.

# Only objects whose MIME type matches one of these patterns are accepted.
# Wildcards supported: "video/*", "image/*", "*". Omit to accept everything.
content_types:
  - image/jpeg
  - image/png
  - image/webp

# Controls whether the raw uploaded file is retained alongside derived artifacts.
# Defaults to true. Set to false to save storage when only derivatives matter.
keep_original: true

# Base filename (without extension) used when the original is saved.
# Defaults to "original". The actual stored name is e.g. "original.jpg".
original_name: original

# Synchronous pre-upload validation (see below).
validate: ...

# Processing DAG (see below).
jobs: ...
```

---

## `validate` block

Validation runs **synchronously** during upload, before any file is persisted. A validation failure is returned immediately to the caller as an error.

```yaml
validate:
  # Maximum allowed file size. Supports KB / MB / GB / TB suffixes.
  max_size: 50MB

  # Minimum allowed file size.
  min_size: 1KB

  # Accepted MIME types for this validation block.
  # Falls back to the top-level content_types when omitted.
  content_types:
    - image/jpeg
    - image/png

  # Additional validation steps executed by registered converters.
  checks:
    - name: check dimensions
      uses: image/check-dimensions
      with:
        min_width: 100
        min_height: 100
```

### `checks` entries

| Field  | Type   | Required | Description                                                  |
| ------ | ------ | -------- | ------------------------------------------------------------ |
| `name` | string | no       | Human-readable label shown in error messages.                |
| `uses` | string | yes      | Converter action identifier (e.g. `image/check-dimensions`). |
| `with` | map    | no       | Key-value parameters forwarded to the converter.             |

---

## `jobs` map

Each key in `jobs` is a job ID. Jobs form a directed acyclic graph (DAG): a job starts only after all its `needs` dependencies have completed.

```yaml
jobs:
  thumbnail: # job ID
    runs-on: cpu # worker affinity (optional)
    needs: [] # upstream job IDs (optional)
    timeout-minutes: 5 # wall-clock timeout; 0 = no timeout (optional)
    on-failure: fail # failure policy (optional; default: fail)
    if: "true" # run condition (optional; default: always run)
    steps:
      - name: Resize to 200px
        uses: image/resize
        with:
          target: thumb.jpg
          width: 200
```

### Job fields

| Field             | Type         | Default | Description                                                                               |
| ----------------- | ------------ | ------- | ----------------------------------------------------------------------------------------- |
| `runs-on`         | string       | `any`   | Worker-affinity label. Accepted values: `any`, `small`, `large`, `gpu`, `label:<custom>`. |
| `needs`           | list[string] | `[]`    | Job IDs that must complete before this job starts.                                        |
| `timeout-minutes` | int          | `0`     | Maximum wall-clock seconds for the job; 0 means no limit.                                 |
| `on-failure`      | string       | `fail`  | Failure policy: `fail`, `continue`, or `retry:N`.                                         |
| `if`              | string       | —       | Expression evaluated before the job runs; job is skipped when false.                      |
| `steps`           | list[Step]   | —       | Ordered actions to execute inside this job.                                               |

### Failure policies

| Value      | Behaviour                                                                                              |
| ---------- | ------------------------------------------------------------------------------------------------------ |
| `fail`     | (default) The pipeline is aborted. Downstream jobs are automatically skipped.                          |
| `continue` | The job is marked `failed` but the pipeline continues. The overall status becomes `partial`.           |
| `retry:N`  | Retry up to N times before treating the job as failed. Uses `fail` semantics after exhausting retries. |

---

## `steps` list

Each step is an action executed in sequence inside its job.

```yaml
steps:
  - name: Convert to WebP # optional label
    uses: image/convert # required action identifier
    with: # optional key-value parameters
      target: preview.webp
      quality: 85
```

| Field  | Type   | Required | Description                                                                       |
| ------ | ------ | -------- | --------------------------------------------------------------------------------- |
| `name` | string | no       | Descriptive label for logs and state.                                             |
| `uses` | string | yes      | Action identifier dispatched to a registered `StepRunner`.                        |
| `with` | map    | no       | Parameters forwarded to the runner. `target` is the conventional output filename. |

---

## `if:` expressions

The `if:` field accepts a simple expression evaluated against the current processing state. The job is **skipped** when the expression evaluates to `false`.

### Supported forms

```yaml
# Literal boolean
if: "true"
if: "false"

# Numeric comparison
if: "${{ probe.outputs.duration }} < 3600"

# Job status check
if: "${{ jobs.thumbnail.status }} == 'completed'"
if: "${{ jobs.probe.status }} != 'failed'"
```

### Expression syntax

| Token      | Examples                                               |
| ---------- | ------------------------------------------------------ |
| Reference  | `${{ jobID.outputs.key }}`, `${{ jobs.jobID.status }}` |
| Comparison | `==`, `!=`, `<`, `<=`, `>`, `>=`                       |
| Literal    | `'completed'`, `'failed'`, `42`, `true`, `false`       |

---

## Worker affinity (`runs-on`)

The `runs-on` value is matched against the worker labels at dispatch time. A job waits until a worker with a matching label picks it up.

| Value         | Meaning                                          |
| ------------- | ------------------------------------------------ |
| `any`         | No preference; any available worker.             |
| `small`       | Lightweight worker (metadata operations, CRC).   |
| `large`       | High-memory worker (image/video processing).     |
| `gpu`         | GPU-capable worker (ML inference, video encode). |
| `label:<tag>` | Custom label; e.g. `label:ffmpeg-6`.             |

---

## ProcessingState JSON

After upload, every object has a `ProcessingState` persisted in `state.json` inside the object directory and exposed via the `Head` RPC.

```json
{
  "object_id": "images/abc123",
  "status": "running",
  "progress": 0.5,
  "manifest_version": "2",
  "started_at": "2026-06-23T10:00:00Z",
  "updated_at": "2026-06-23T10:00:05Z",
  "finished_at": null,
  "jobs": {
    "thumbnail": {
      "status": "completed",
      "worker": "cpu-node-2",
      "attempts": 1,
      "outputs": { "path": "thumb.jpg", "width": 200 },
      "steps": [
        { "name": "Resize to 200px", "status": "completed", "duration_ms": 48 }
      ],
      "started_at": "2026-06-23T10:00:01Z",
      "finished_at": "2026-06-23T10:00:01Z"
    },
    "medium": {
      "status": "running",
      "worker": "cpu-node-1",
      "steps": [
        { "name": "Resize to 800px", "status": "running", "duration_ms": 0 }
      ]
    },
    "large": {
      "status": "pending"
    }
  }
}
```

### Top-level status values

| Status      | Meaning                                                                       |
| ----------- | ----------------------------------------------------------------------------- |
| `pending`   | No jobs have started yet.                                                     |
| `running`   | At least one job is currently executing.                                      |
| `completed` | All jobs finished without failures.                                           |
| `partial`   | All jobs finished; at least one failed with `on-failure: continue`.           |
| `failed`    | At least one job failed with `on-failure: fail` and the pipeline was aborted. |

---

## Complete example

```yaml
version: "2"
name: "Image gallery pipeline"
content_types:
  - image/jpeg
  - image/png
  - image/webp
keep_original: true

validate:
  max_size: 20MB
  min_size: 1KB
  content_types: [image/jpeg, image/png, image/webp]

jobs:
  thumbnail:
    runs-on: small
    steps:
      - name: Generate thumbnail
        uses: image/resize
        with:
          target: thumb.jpg
          width: 200
          height: 200
          fit: cover

  medium:
    runs-on: small
    steps:
      - name: Generate medium
        uses: image/resize
        with:
          target: medium.jpg
          width: 800

  large:
    runs-on: large
    steps:
      - name: Generate large
        uses: image/resize
        with:
          target: large.jpg
          width: 1920

  blur-preview:
    runs-on: small
    needs: [thumbnail]
    on-failure: continue
    steps:
      - name: Blur thumbnail
        uses: image/blur
        with:
          target: blur.jpg
          source: thumb.jpg
          radius: 20
```
