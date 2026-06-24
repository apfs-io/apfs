# APFS — Autoprocessing File System

[![Build Status](https://github.com/apfs-io/apfs/workflows/Tests/badge.svg)](https://github.com/apfs-io/apfs/actions?workflow=Tests)
[![Go Report Card](https://goreportcard.com/badge/github.com/apfs-io/apfs)](https://goreportcard.com/report/github.com/apfs-io/apfs)
[![GoDoc](https://godoc.org/github.com/apfs-io/apfs?status.svg)](https://godoc.org/github.com/apfs-io/apfs)
[![Coverage Status](https://coveralls.io/repos/github/apfs-io/apfs/badge.svg)](https://coveralls.io/github/apfs-io/apfs)

`apfs-io/apfs` is an automated file-processing system built for high-performance, pipeline-driven handling of complex objects. Upload a file and a declarative workflow takes over: validating, transforming, and persisting every derived artifact while tracking progress in real time.

## Features

- **Declarative workflows** — GitHub Actions-inspired YAML defines your processing pipeline per bucket.
- **Pre-upload validation** — synchronous size and content-type checks before any file is stored.
- **DAG execution** — jobs declare `needs:` dependencies; independent jobs run in parallel on matched workers.
- **Conditional logic** — `if:` expressions skip jobs based on upstream outputs or statuses.
- **Failure policies** — per-job `on-failure: fail | continue | retry:N` controls pipeline behaviour.
- **Processing state** — every object carries a `ProcessingState` (progress 0–1, per-job/step statuses).
- **Complex objects** — a single upload may produce many derived files (`thumb.jpg`, `720p.mp4`, …).
- **gRPC + REST gateway** — Protocol Buffers API with an auto-generated REST facade.
- **Pluggable storage drivers** — local filesystem and S3-compatible backends out of the box.

## Quick Start

```go
import (
    "github.com/apfs-io/apfs/libs/client"
    "github.com/apfs-io/apfs/models"
)

cl, _ := client.Connect("localhost:8080")
videos := cl.Group("videos")

// 1. Define the processing workflow for the bucket (once per deployment).
wf := &models.Workflow{
    Version:      "2",
    ContentTypes: []string{"video/*"},
    Validate: &models.WorkflowValidate{
        MaxSize:      "2GB",
        ContentTypes: []string{"video/mp4", "video/quicktime"},
    },
    Jobs: map[string]*models.WorkflowJob{
        "thumbnail": {
            RunsOn: "cpu",
            Steps: []*models.WorkflowStep{
                {Uses: "ffmpeg/thumbnail", With: map[string]any{"target": "thumb.jpg", "time": "00:00:02"}},
            },
        },
        "transcode-720p": {
            RunsOn: "gpu",
            Needs:  []string{"thumbnail"},
            Steps: []*models.WorkflowStep{
                {Uses: "ffmpeg/encode", With: map[string]any{"target": "720p.mp4", "resolution": "1280x720"}},
            },
        },
    },
}
videos.SetWorkflow(ctx, wf)

// 2. Upload a file — validation runs synchronously, processing starts asynchronously.
f, _ := os.Open("promo.mp4")
obj, _ := videos.Upload(ctx, f, client.WithTags("promo"))

// 3. Poll progress.
videos.WatchProgress(ctx, obj.ID, func(state *models.ProcessingState) {
    fmt.Printf("progress: %.0f%%  status: %s\n", state.Progress*100, state.Status)
})
```

See [docs/USE_CASES.md](docs/USE_CASES.md) for complete examples and [docs/WORKFLOW.md](docs/WORKFLOW.md) for the full schema reference.

## API Overview

### Services

1. **Object operations**
   - `Head` — retrieve object metadata and `ProcessingState`.
   - `Get` — fetch an object's data stream and metadata.
   - `Refresh` — trigger re-processing of an existing object.
   - `Delete` — remove an object or specific sub-files.

2. **Workflow management**
   - `SetWorkflow` — store or update the processing workflow for a bucket.
   - `GetWorkflow` — retrieve the current workflow for a bucket.

3. **Data upload**
   - `Upload` — stream a new file into the system; pre-upload validation runs before persistence.

### Processing State

Every object carries a `ProcessingState` describing the execution of its workflow:

```json
{
  "object_id": "videos/abc123",
  "status": "running",
  "progress": 0.33,
  "jobs": {
    "thumbnail": { "status": "completed", "outputs": { "path": "thumb.jpg" } },
    "transcode-720p": { "status": "running", "worker": "gpu-node-1" },
    "transcode-480p": { "status": "pending" }
  }
}
```

Possible top-level statuses: `pending`, `running`, `completed`, `partial` (some jobs failed with `on-failure: continue`), `failed`.

### REST Endpoints

| Method   | Endpoint               | Description                             |
| -------- | ---------------------- | --------------------------------------- |
| `GET`    | `/v1/head/{id}`        | Retrieve object metadata.               |
| `GET`    | `/v1/object/{id}`      | Retrieve object and data stream.        |
| `PUT`    | `/v1/refresh/{id}`     | Trigger re-processing of an object.     |
| `PUT`    | `/v1/manifest/{group}` | Set the workflow for a bucket.          |
| `GET`    | `/v1/manifest/{group}` | Retrieve the workflow for a bucket.     |
| `POST`   | `/v1/object`           | Upload a new file.                      |
| `DELETE` | `/v1/object/{id}`      | Delete an object or specific sub-files. |

### Protocol Buffers

The API is defined with `proto3` in [`protocol/v1/`](protocol/v1/). A REST gateway is generated via [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway). OpenAPI specs are available for easy client generation.

## Documentation

| Document                                                     | Description                                                   |
| ------------------------------------------------------------ | ------------------------------------------------------------- |
| [docs/WORKFLOW.md](docs/WORKFLOW.md)                         | Full v2 workflow YAML schema reference                        |
| [docs/USE_CASES.md](docs/USE_CASES.md)                       | End-to-end examples: image gallery, video, documents, avatars |
| [internal/driver/s3/README.md](internal/driver/s3/README.md) | Local S3 setup with MinIO                                     |

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
