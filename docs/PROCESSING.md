# File Processing — Status Tracking

This document describes how to track the progress of APFS object processing:
the status lifecycle, real-time event stream subscription, and polling via the
gRPC API.

See also: [WORKFLOW.md](WORKFLOW.md) (how processing pipelines are defined),
[USE_CASES.md](USE_CASES.md) (end-to-end upload + processing examples).

---

## Processing status lifecycle

Every object has a `ProcessingStatus` that advances through the following states:

| Status      | Terminal | Success | Meaning                                               |
| ----------- | :------: | :-----: | ----------------------------------------------------- |
| `pending`   |    no    |    —    | Uploaded; waiting for a worker to pick it up          |
| `running`   |    no    |    —    | One or more pipeline jobs are currently executing     |
| `completed` | **yes**  | **yes** | All jobs finished successfully                        |
| `partial`   | **yes**  | **yes** | All jobs ran; some with `on-failure: continue` failed |
| `failed`    | **yes**  |   no    | At least one required job failed                      |

```
pending ──► running ──► completed
                   └──► partial
                   └──► failed
```

Helper methods on `models.ProcessingStatus`:

```go
status.IsTerminal() // true for completed, partial, failed
status.IsSuccess()  // true for completed, partial
```

---

## Option 1 — Real-time stream subscription

APFS publishes a `ProcessingStatusEvent` to the configured status stream after
every task completion and as a final summary event once all pipeline jobs finish.

### Event structure

```go
// libs/client.ProcessingStatusEvent (alias of ctxstatusstream.ProcessingStatusEvent)
type ProcessingStatusEvent struct {
    ObjectID  string                  // which object this event belongs to
    Status    models.ProcessingStatus // current pipeline status (see lifecycle above)
    Progress  float64                 // 0.0–1.0 completion ratio
    Total     int                     // total tasks in the pipeline
    Completed int                     // tasks that finished successfully
    Failed    int                     // tasks that finished with an error
    Skipped   int                     // tasks skipped (on-failure: continue)
    Pending   int                     // tasks not yet started
    Error     string                  // non-empty when Status == "failed"
    Final     bool                    // true on the last event for this object
}
```

`Final` is set on the single summary event emitted after all tasks are done.
Intermediate events (`Final == false`) carry counters and `Status == "running"`.

### Quick subscription with `SubscribeProcessing`

The simplest way to consume the stream — connects, subscribes, and blocks until
the context is cancelled.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/apfs-io/apfs/libs/client"
)

func main() {
    ctx := context.Background()

    err := client.SubscribeProcessing(ctx, "nats://nats:4222/apfs?topics=status",
        func(ctx context.Context, e *client.ProcessingStatusEvent) error {
            fmt.Printf("object=%-36s  status=%-10s  progress=%.0f%%  jobs=%d/%d",
                e.ObjectID, e.Status, e.Progress*100, e.Completed, e.Total)
            if e.Final {
                fmt.Printf("  [FINAL]")
                if e.Error != "" {
                    fmt.Printf("  err=%s", e.Error)
                }
            }
            fmt.Println()
            return nil
        })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Manual `ProcessingStream` with graceful shutdown

Use `ProcessingStream` directly when you need to decouple subscription from
listening or to share the stream across multiple handlers.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/apfs-io/apfs/libs/client"
    "github.com/apfs-io/apfs/models"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(),
        os.Interrupt, syscall.SIGTERM)
    defer cancel()

    ps, err := client.NewProcessingStream(ctx, "nats://nats:4222/apfs?topics=status")
    if err != nil {
        log.Fatal(err)
    }
    defer ps.Close()

    if err := ps.Subscribe(ctx, handleEvent); err != nil {
        log.Fatal(err)
    }

    // blocks until ctx is cancelled
    if err := ps.Listen(ctx); err != nil && err != context.Canceled {
        log.Fatal(err)
    }
}

func handleEvent(ctx context.Context, e *client.ProcessingStatusEvent) error {
    if !e.Final {
        // Intermediate progress update — only update UI / progress bar.
        fmt.Printf("%s: running  %.0f%%\n", e.ObjectID, e.Progress*100)
        return nil
    }

    switch {
    case e.Status.IsSuccess():
        fmt.Printf("%s: done  completed=%d skipped=%d\n",
            e.ObjectID, e.Completed, e.Skipped)
    case e.Status == models.ProcessingStatusFailed:
        fmt.Printf("%s: FAILED  err=%s\n", e.ObjectID, e.Error)
    }
    return nil
}
```

### Routing events to per-object handlers

A common pattern when integrating with a background worker that processes many
objects concurrently:

```go
type Tracker struct {
    mu       sync.Mutex
    handlers map[string]func(*client.ProcessingStatusEvent)
}

func (t *Tracker) Register(objectID string, h func(*client.ProcessingStatusEvent)) {
    t.mu.Lock()
    t.handlers[objectID] = h
    t.mu.Unlock()
}

func (t *Tracker) Handle(ctx context.Context, e *client.ProcessingStatusEvent) error {
    t.mu.Lock()
    h, ok := t.handlers[e.ObjectID]
    if ok && e.Final {
        delete(t.handlers, e.ObjectID)
    }
    t.mu.Unlock()
    if ok {
        h(e)
    }
    return nil
}

// wiring
tracker := &Tracker{handlers: map[string]func(*client.ProcessingStatusEvent){}}

go client.SubscribeProcessing(ctx, streamURL, tracker.Handle) //nolint:errcheck

// later, when uploading:
obj, _ := group.Upload(ctx, f)
done := make(chan *client.ProcessingStatusEvent, 1)
tracker.Register(obj.ID, func(e *client.ProcessingStatusEvent) { done <- e })
result := <-done
fmt.Println("status:", result.Status)
```

---

## Option 2 — Polling via `WatchProgress`

When a real-time stream is not available, use `Group.WatchProgress`. It polls
`ProcessingState` until the object reaches a terminal status or the context is
cancelled.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/apfs-io/apfs/libs/client"
)

func main() {
    ctx := context.Background()
    cl, err := client.Connect(ctx, "tcp://localhost:8080/images")
    if err != nil {
        log.Fatal(err)
    }

    images := cl.Group("images")

    f, _ := os.Open("photo.jpg")
    defer f.Close()

    obj, err := images.Upload(ctx, f, client.WithTags("gallery"))
    if err != nil {
        log.Fatal("upload:", err)
    }
    fmt.Println("uploaded:", obj.ID)

    // Poll every ~1 s (default interval) until terminal state.
    if err := images.WatchProgress(ctx, obj.ID, func(state *client.ProcessingState) {
        fmt.Printf("  progress=%.0f%%  status=%s  jobs=%d/%d\n",
            state.Progress*100,
            state.Status,
            state.Counters.Succeeded,
            state.Counters.Total,
        )
    }); err != nil {
        log.Fatal("watch:", err)
    }

    fmt.Println("done!")
}
```

### One-shot state query

Use `Group.ProcessingState` when you only need the current snapshot:

```go
state, err := images.ProcessingState(ctx, objectID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("status=%s  progress=%.0f%%\n", state.Status, state.Progress*100)

if state.Status.IsTerminal() {
    if state.Status.IsSuccess() {
        fmt.Println("processing finished successfully")
    } else {
        fmt.Println("processing failed")
    }
}
```

---

## Option 3 — Full per-job detail

Pass `client.WithFullState()` to get per-job and per-step breakdowns:

```go
obj, err := images.Head(ctx, objectID, client.WithFullState())
if err != nil {
    log.Fatal(err)
}
state := obj.State

for name, job := range state.Jobs {
    fmt.Printf("job %-20s  status=%-10s  attempts=%d\n",
        name, job.Status, job.Attempts)
    for _, step := range job.Steps {
        fmt.Printf("  step %-18s  status=%-10s  duration=%dms\n",
            step.Name, step.Status, step.DurationMs)
    }
    if job.Error != "" {
        fmt.Printf("  error: %s\n", job.Error)
    }
}
```

---

## APFS server-side stream publishing

APFS publishes events via `ctxstatusstream.Publish` from within the processor
after every task and at pipeline completion (v1 `ProcessTasks` and v2
`ExecuteJob` / `ProcessObject`). Configure the stream publisher:

```yaml
processing:
  status_stream:
    connect: "nats://nats:4222/apfs?topics=status"
  stall_check_interval: 2m
```

Env: `PROCESSING_STATUS_STREAM_CONNECT`, `PROCESSING_STALL_CHECK_INTERVAL`
(`0` disables the stall watchdog).

### `EnsureProcessing`

Clients (and the adnetapi watchdog) can ask APFS to check an object:

```go
state, err := cl.EnsureProcessing(ctx, apfs.ID(objectID))
```

- Always republishes the current `ProcessingStatusEvent` on the status stream.
- If the object is **not** terminal, enqueues another `update` work event.
- If the object **is** terminal, publishes `Final: true` and drops it from the
  in-flight registry.

APFS also keeps an in-flight registry (memory or Redis via `STORAGE_STATE_CONNECT`)
and a stall watchdog that runs `EnsureProcessing` for objects older than
`PROCESSING_STALL_CHECK_INTERVAL`.

| Trigger           | `Status`                           | `Final`  | Notes                                                                 |
| ----------------- | ---------------------------------- | :------: | --------------------------------------------------------------------- |
| Task completed    | `running`                          |  false   | Per-task progress event; `Completed` incremented                      |
| Task failed       | `failed`                           |  false   | `Failed` incremented; pipeline may continue if `on-failure: continue` |
| Pipeline finished | `completed` / `partial` / `failed` | **true** | Summary event with final counters and optional `Error`                |

### Event `Progress` calculation

`Progress` is a `float64` in the range `0.0–1.0`:

- `1.0` — all tasks completed without error (`completed`)
- `failed / total` error cases — ratio of completed to total tasks
- `partial` — total completed / total (skipped tasks count towards completed ratio)

---

## Choosing between stream and polling

| Criterion                   | Stream (`SubscribeProcessing`)   | Polling (`WatchProgress`)      |
| --------------------------- | -------------------------------- | ------------------------------ |
| Latency                     | Near real-time                   | Depends on poll interval       |
| Infrastructure requirement  | Message broker (NATS, etc.)      | gRPC only                      |
| Suitable for many objects   | Yes — single connection, fan-out | Yes — one goroutine per object |
| Suitable for one-off checks | Overkill                         | Ideal                          |
| Survives broker restart     | Requires reconnect logic         | Always available               |
| Per-object routing          | Manual (see routing example)     | Built-in (WatchProgress)       |
