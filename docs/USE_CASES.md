# Use Cases

Each section shows a complete workflow YAML and the Go client code needed to register it and upload a file.

---

## 1. Image gallery

**Scenario:** A user uploads a JPEG or PNG photo. The system validates it, then generates three resized versions (`thumb.jpg`, `medium.jpg`, `large.jpg`) and a blurred preview (`blur.jpg`). All four derivatives are created in parallel. The original is retained.

### Workflow (`manifest.yaml`)

```yaml
version: "2"
name: "Image gallery"
content_types: [image/jpeg, image/png, image/webp]
keep_original: true

validate:
  max_size: 20MB
  min_size: 1KB
  content_types: [image/jpeg, image/png, image/webp]

jobs:
  thumbnail:
    runs-on: small
    steps:
      - name: Thumbnail 200px
        uses: image/resize
        with: { target: thumb.jpg, width: 200, height: 200, fit: cover }

  medium:
    runs-on: small
    steps:
      - name: Medium 800px
        uses: image/resize
        with: { target: medium.jpg, width: 800 }

  large:
    runs-on: large
    steps:
      - name: Large 1920px
        uses: image/resize
        with: { target: large.jpg, width: 1920 }

  blur-preview:
    runs-on: small
    needs: [thumbnail]
    on-failure: continue
    steps:
      - name: Blurred preview
        uses: image/blur
        with: { target: blur.jpg, source: thumb.jpg, radius: 20 }
```

### Go client

```go
package main

import (
    "context"
    "os"

    "github.com/apfs-io/apfs/libs/client"
    "github.com/apfs-io/apfs/models"
)

func main() {
    ctx := context.Background()
    cl, _ := client.Connect("localhost:8080")
    photos := cl.Group("photos")

    wf := &models.Workflow{
        Version:      "2",
        ContentTypes: []string{"image/jpeg", "image/png", "image/webp"},
        Validate: &models.WorkflowValidate{
            MaxSize:      "20MB",
            ContentTypes: []string{"image/jpeg", "image/png", "image/webp"},
        },
        Jobs: map[string]*models.WorkflowJob{
            "thumbnail": {RunsOn: "small", Steps: []*models.WorkflowStep{
                {Uses: "image/resize", With: map[string]any{"target": "thumb.jpg", "width": 200, "height": 200, "fit": "cover"}},
            }},
            "medium": {RunsOn: "small", Steps: []*models.WorkflowStep{
                {Uses: "image/resize", With: map[string]any{"target": "medium.jpg", "width": 800}},
            }},
            "large": {RunsOn: "large", Steps: []*models.WorkflowStep{
                {Uses: "image/resize", With: map[string]any{"target": "large.jpg", "width": 1920}},
            }},
            "blur-preview": {
                RunsOn: "small", Needs: []string{"thumbnail"}, OnFailure: "continue",
                Steps: []*models.WorkflowStep{
                    {Uses: "image/blur", With: map[string]any{"target": "blur.jpg", "source": "thumb.jpg", "radius": 20}},
                },
            },
        },
    }
    _ = photos.SetWorkflow(ctx, wf)

    f, _ := os.Open("photo.jpg")
    defer f.Close()
    obj, _ := photos.Upload(ctx, f, client.WithTags("gallery", "user-upload"))

    _ = photos.WatchProgress(ctx, obj.ID, func(state *models.ProcessingState) {
        // called each poll cycle until terminal state
        println("progress:", state.Progress, "status:", string(state.Status))
    })
}
```

---

## 2. Video transcoding

**Scenario:** A video is uploaded, validated for size and MIME type, and then processed in two stages. First a thumbnail is extracted on a CPU worker. Once that finishes, three resolutions are transcoded in parallel on GPU workers. The original is discarded after processing.

### Workflow (`manifest.yaml`)

```yaml
version: "2"
name: "Video transcoding"
content_types: [video/mp4, video/quicktime, video/x-matroska]
keep_original: false
original_name: source

validate:
  max_size: 2GB
  min_size: 100KB
  content_types: [video/mp4, video/quicktime, video/x-matroska]

jobs:
  # Stage 1 — runs on any CPU worker
  thumbnail:
    runs-on: cpu
    timeout-minutes: 2
    steps:
      - name: Extract frame at 2s
        uses: ffmpeg/thumbnail
        with: { target: thumb.jpg, time: "00:00:02" }

  # Stage 2 — all three run in parallel after thumbnail is done
  transcode-1080p:
    runs-on: gpu
    needs: [thumbnail]
    timeout-minutes: 60
    on-failure: continue
    steps:
      - name: Encode 1080p
        uses: ffmpeg/encode
        with: { target: 1080p.mp4, resolution: "1920x1080", crf: 23 }

  transcode-720p:
    runs-on: gpu
    needs: [thumbnail]
    timeout-minutes: 40
    on-failure: continue
    steps:
      - name: Encode 720p
        uses: ffmpeg/encode
        with: { target: 720p.mp4, resolution: "1280x720", crf: 25 }

  transcode-480p:
    runs-on: gpu
    needs: [thumbnail]
    timeout-minutes: 20
    on-failure: continue
    steps:
      - name: Encode 480p
        uses: ffmpeg/encode
        with: { target: 480p.mp4, resolution: "854x480", crf: 28 }
```

### Go client

```go
videos := cl.Group("videos")

wf := &models.Workflow{
    Version:      "2",
    ContentTypes: []string{"video/mp4", "video/quicktime"},
    KeepOriginal: boolPtr(false),
    Validate: &models.WorkflowValidate{
        MaxSize: "2GB",
        MinSize: "100KB",
    },
    Jobs: map[string]*models.WorkflowJob{
        "thumbnail": {
            RunsOn: "cpu", TimeoutMinutes: 2,
            Steps: []*models.WorkflowStep{
                {Uses: "ffmpeg/thumbnail", With: map[string]any{"target": "thumb.jpg", "time": "00:00:02"}},
            },
        },
        "transcode-1080p": {
            RunsOn: "gpu", Needs: []string{"thumbnail"}, OnFailure: "continue", TimeoutMinutes: 60,
            Steps: []*models.WorkflowStep{
                {Uses: "ffmpeg/encode", With: map[string]any{"target": "1080p.mp4", "resolution": "1920x1080"}},
            },
        },
        "transcode-720p": {
            RunsOn: "gpu", Needs: []string{"thumbnail"}, OnFailure: "continue", TimeoutMinutes: 40,
            Steps: []*models.WorkflowStep{
                {Uses: "ffmpeg/encode", With: map[string]any{"target": "720p.mp4", "resolution": "1280x720"}},
            },
        },
        "transcode-480p": {
            RunsOn: "gpu", Needs: []string{"thumbnail"}, OnFailure: "continue", TimeoutMinutes: 20,
            Steps: []*models.WorkflowStep{
                {Uses: "ffmpeg/encode", With: map[string]any{"target": "480p.mp4", "resolution": "854x480"}},
            },
        },
    },
}
_ = videos.SetWorkflow(ctx, wf)

f, _ := os.Open("lecture.mp4")
obj, _ := videos.Upload(ctx, f)
```

---

## 3. Document archive

**Scenario:** PDFs are uploaded and archived. Three operations run sequentially: first a CRC32 checksum is computed (used for deduplication), then text is extracted for search indexing, then a preview thumbnail of the cover page is rendered. All steps use `on-failure: continue` so that one tool failure does not block the others.

### Workflow (`manifest.yaml`)

```yaml
version: "2"
name: "Document archive"
content_types: [application/pdf]
keep_original: true

validate:
  max_size: 100MB
  content_types: [application/pdf]

jobs:
  checksum:
    runs-on: small
    on-failure: continue
    steps:
      - name: Compute CRC32
        uses: file/crc32
        with: { target: checksum.txt }

  extract-text:
    runs-on: small
    needs: [checksum]
    on-failure: continue
    steps:
      - name: Extract text
        uses: pdf/extract-text
        with: { target: content.txt }

  cover-thumbnail:
    runs-on: large
    needs: [checksum]
    on-failure: continue
    steps:
      - name: Render cover page
        uses: pdf/thumbnail
        with: { target: cover.jpg, page: 0, width: 400 }
```

### Go client

```go
docs := cl.Group("documents")

wf := &models.Workflow{
    Version:      "2",
    ContentTypes: []string{"application/pdf"},
    Validate:     &models.WorkflowValidate{MaxSize: "100MB"},
    Jobs: map[string]*models.WorkflowJob{
        "checksum": {
            RunsOn: "small", OnFailure: "continue",
            Steps: []*models.WorkflowStep{
                {Uses: "file/crc32", With: map[string]any{"target": "checksum.txt"}},
            },
        },
        "extract-text": {
            RunsOn: "small", Needs: []string{"checksum"}, OnFailure: "continue",
            Steps: []*models.WorkflowStep{
                {Uses: "pdf/extract-text", With: map[string]any{"target": "content.txt"}},
            },
        },
        "cover-thumbnail": {
            RunsOn: "large", Needs: []string{"checksum"}, OnFailure: "continue",
            Steps: []*models.WorkflowStep{
                {Uses: "pdf/thumbnail", With: map[string]any{"target": "cover.jpg", "page": 0, "width": 400}},
            },
        },
    },
}
_ = docs.SetWorkflow(ctx, wf)

f, _ := os.Open("report.pdf")
obj, _ := docs.Upload(ctx, f, client.WithTags("annual-report", "2026"))
fmt.Println("uploaded:", obj.ID)
```

---

## 4. User avatar

**Scenario:** A user uploads any image as their avatar. The image is validated for MIME type, resized to a square 256×256 thumbnail, and the original is discarded — only the processed avatar is kept.

### Workflow (`manifest.yaml`)

```yaml
version: "2"
name: "User avatar"
content_types: [image/jpeg, image/png, image/gif, image/webp]
keep_original: false

validate:
  max_size: 5MB
  content_types: [image/jpeg, image/png, image/gif, image/webp]

jobs:
  avatar:
    runs-on: small
    steps:
      - name: Resize to 256x256
        uses: image/resize
        with:
          target: avatar.jpg
          width: 256
          height: 256
          fit: cover
          format: jpeg
          quality: 90
```

### Go client

```go
avatars := cl.Group("avatars")

falseVal := false
wf := &models.Workflow{
    Version:      "2",
    ContentTypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
    KeepOriginal: &falseVal,
    Validate: &models.WorkflowValidate{
        MaxSize:      "5MB",
        ContentTypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
    },
    Jobs: map[string]*models.WorkflowJob{
        "avatar": {
            RunsOn: "small",
            Steps: []*models.WorkflowStep{
                {
                    Uses: "image/resize",
                    With: map[string]any{
                        "target":  "avatar.jpg",
                        "width":   256,
                        "height":  256,
                        "fit":     "cover",
                        "format":  "jpeg",
                        "quality": 90,
                    },
                },
            },
        },
    },
}
_ = avatars.SetWorkflow(ctx, wf)

// Upload from an HTTP multipart request.
// client.WithCustomID pins the object to the user's ID.
obj, err := avatars.Upload(ctx, r.Body,
    client.WithCustomID(userID),
    client.WithTags("avatar"),
)
if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest) // validation failures propagate here
    return
}
fmt.Fprintln(w, "avatar stored:", obj.ID)
```

---

## Key patterns

| Pattern | How |
|---------|-----|
| Parallel processing | Omit `needs:` — jobs with no dependencies start immediately and run concurrently. |
| Sequential stages | Use `needs: [previous-job]` to enforce ordering. |
| Best-effort steps | Set `on-failure: continue` on non-critical jobs; the overall status becomes `partial`. |
| Conditional logic | Use `if: "${{ jobs.probe.status }} == 'completed'"` to skip expensive jobs when a probe step fails. |
| Discard original | Set `keep_original: false`; only derived files are retained. |
| Retry transient errors | Set `on-failure: retry:3`; the worker re-attempts up to three times before failing. |
| Worker affinity | Use `runs-on: gpu` to route expensive jobs to capable nodes without changing the workflow logic. |
