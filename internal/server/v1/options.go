package v1

import (
	"github.com/apfs-io/apfs/internal/storage"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	"github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/internal/workflow"

	nc "github.com/geniusrabbit/notificationcenter/v2"
)

// Option update function
type Option func(*Options)

// Options of the server
type Options struct {
	// Task Processing limit (max workflow jobs per Update event)
	taskProcessingLimit int

	// Storage object
	store *storage.Storage

	// Event stream object chanel
	eventStream nc.Publisher

	// Status stream for per-task progress events (optional)
	statusStream nc.Publisher

	// Update state accessor
	updateState updateStateI

	// Workflow runner registry (optional; jobless workflows still complete)
	wfRegistry *workflow.RunnerRegistry

	// Worker tags for workflow job affinity
	workerTags []string

	// Workflows bootstrap from filesystem on startup
	workflowsDir         string
	workflowsReconfigure bool

	// ensureBucket nil = S3 driver default (true).
	ensureBucket *bool
}

func (opts *Options) _storage(database storage.DB, driver storio.StorageAccessor, stateKV kvaccessor.KVAccessor) *storage.Storage {
	if opts.store == nil {
		opts.store = storage.NewStorage(
			storage.WithDatabase(database),
			storage.WithDriver(driver),
			storage.WithProcessingStatus(stateKV),
		)
	}
	return opts.store
}

// WithTaskProcessingLimit custom option
func WithTaskProcessingLimit(limit int) Option {
	return func(opts *Options) {
		opts.taskProcessingLimit = limit
	}
}

// WithStorage custom option
func WithStorage(store *storage.Storage) Option {
	return func(opts *Options) {
		opts.store = store
	}
}

// WithEventstream cannel option
func WithEventstream(eventStream nc.Publisher) Option {
	return func(opts *Options) {
		opts.eventStream = eventStream
	}
}

// WithStatusStream sets the optional publisher used for per-task progress events.
func WithStatusStream(pub nc.Publisher) Option {
	return func(opts *Options) {
		opts.statusStream = pub
	}
}

// WithUpdateState memory checkpoint option
func WithUpdateState(updateState updateStateI) Option {
	return func(opts *Options) {
		opts.updateState = updateState
	}
}

// WithWorkflowExecutor registers workflow step runners for the event pipeline.
func WithWorkflowExecutor(registry *workflow.RunnerRegistry) Option {
	return func(opts *Options) {
		opts.wfRegistry = registry
	}
}

// WithWorkerTags sets worker capability tags used for workflow runs-on matching.
func WithWorkerTags(tags []string) Option {
	return func(opts *Options) {
		opts.workerTags = tags
	}
}

// WithWorkflowsBootstrap seeds bucket workflows from a directory on startup.
func WithWorkflowsBootstrap(dir string, reconfigure bool) Option {
	return func(opts *Options) {
		opts.workflowsDir = dir
		opts.workflowsReconfigure = reconfigure
	}
}

// WithEnsureBucket controls S3 HeadBucket/CreateBucket of the DSN bucket.
func WithEnsureBucket(ensure bool) Option {
	return func(opts *Options) {
		v := ensure
		opts.ensureBucket = &v
	}
}
