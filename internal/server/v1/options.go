package v1

import (
	"github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/storage"
	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	"github.com/apfs-io/apfs/internal/storage/processor"

	nc "github.com/geniusrabbit/notificationcenter/v2"
)

// Option update function
type Option func(*Options)

// Options of the server
type Options struct {
	// Amount of stages processed per one iteration
	stageProcessingLimit int

	// Task Processing limit
	taskProcessingLimit int

	// Max amount of attempts of the task processing
	maxRetries int

	// Storage object
	store *storage.Storage

	// Converter drivers
	convs []converters.Converter

	// Event stream object chanel
	eventStream nc.Publisher

	// Update state accessor
	updateState updateStateI
}

func (opts *Options) _storage(database storage.DB, driver io.StorageAccessor, stateKV kvaccessor.KVAccessor) *storage.Storage {
	if opts.store == nil {
		opts.store = storage.NewStorage(
			storage.WithDatabase(database),
			storage.WithDriver(driver),
			storage.WithProcessingStatus(stateKV),
		)
	}
	return opts.store
}

func (opts *Options) _processor(driver io.StorageAccessor, stateKV kvaccessor.KVAccessor) *processor.Processor {
	proc, err := processor.NewProcessor(
		processor.WithStorage(opts.store),
		processor.WithDriver(driver),
		processor.WithProcessingStatus(stateKV),
		processor.WithConverters(opts.convs...),
		processor.WithStorage(opts.store),
		processor.WithMaxRetries(opts.maxRetries),
	)
	if err != nil {
		panic(err)
	}
	return proc
}

// WithStageProcessingLimit custom option
func WithStageProcessingLimit(limit int) Option {
	return func(opts *Options) {
		opts.stageProcessingLimit = limit
	}
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

// WithStorageConverters custom option
func WithStorageConverters(convs []converters.Converter) Option {
	return func(opts *Options) {
		opts.convs = convs
	}
}

// WithEventstream cannel option
func WithEventstream(eventStream nc.Publisher) Option {
	return func(opts *Options) {
		opts.eventStream = eventStream
	}
}

// WithUpdateState memory checkpoint option
func WithUpdateState(updateState updateStateI) Option {
	return func(opts *Options) {
		opts.updateState = updateState
	}
}

// WithRetries count of attempts
func WithRetries(maxRetries int) Option {
	return func(opts *Options) {
		opts.maxRetries = maxRetries
	}
}
