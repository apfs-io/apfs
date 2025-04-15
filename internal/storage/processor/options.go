package processor

import (
	"errors"

	"github.com/demdxx/xtypes"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
)

var (
	ErrStorageDriverIsRequired = errors.New("storage driver is required")
)

// Options of the storage
type Options struct {
	// storage connector interface
	storage Storage

	// collection of file objects
	driver npio.StorageAccessor

	// converter list of processing
	converters []converters.Converter

	// Processing status KeyValue accessor.
	// contains statuses of the object processing stages
	processingStatus kvaccessor.KVAccessor

	// MaxRetries of the task processing
	maxRetries int
}

func (opts *Options) validate() error {
	if opts.driver == nil {
		return ErrStorageDriverIsRequired
	}
	return nil
}

// Option contains basic storage properties
type Option func(opts *Options)

// WithStorage connector interface
func WithStorage(storage Storage) Option {
	return func(opts *Options) {
		opts.storage = storage
	}
}

// WithDriver object accessor interface
func WithDriver(driver npio.StorageAccessor) Option {
	return func(opts *Options) {
		opts.driver = driver
	}
}

// WithProcessingStatus object status accessor interface
func WithProcessingStatus(processingStatus kvaccessor.KVAccessor) Option {
	return func(opts *Options) {
		opts.processingStatus = processingStatus
	}
}

// WithConverters list of processors
func WithConverters(converters ...converters.Converter) Option {
	return func(opts *Options) {
		opts.converters = xtypes.SliceUnique(append(opts.converters, converters...))
	}
}

// WithMaxRetries of the task execution
func WithMaxRetries(maxRetries int) Option {
	return func(opts *Options) {
		opts.maxRetries = maxRetries
	}
}
