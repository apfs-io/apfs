package storage

import (
	"net/url"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
)

type uploadOption struct {
	customID  npio.ObjectID
	tags      []string
	params    map[string][]string
	overwrite bool
}

func (opt *uploadOption) Params() url.Values {
	params := make(url.Values, 10)
	if opt.params != nil {
		for k, v := range opt.params {
			params[k] = v
		}
	}
	if len(opt.tags) > 0 {
		params["tags"] = opt.tags
	}
	return params
}

// UploadOption type
type UploadOption func(opt *uploadOption)

// WithTags of the file upload
func WithTags(tags []string) UploadOption {
	return func(opt *uploadOption) {
		opt.tags = tags
	}
}

// WithParams of the file upload
func WithParams(params map[string][]string) UploadOption {
	return func(opt *uploadOption) {
		opt.params = params
	}
}

// WithOverwrite of the custom file
func WithOverwrite(overwrite bool) UploadOption {
	return func(opt *uploadOption) {
		opt.overwrite = overwrite
	}
}

// WithOverwrite of the custom file
func WithCustomID(id npio.ObjectID) UploadOption {
	return func(opt *uploadOption) {
		if id != nil && id.ID().String() == "" {
			id = nil
		}
		opt.customID = id
	}
}

// Options of the storage
type Options struct {
	// database storage accessor
	Database DB

	// collection of file objects
	Driver npio.StorageAccessor

	// converter list of processing
	Converters []converters.Converter

	// Processing status KeyValue accessor.
	// contains statuses of the object processing stages
	ProcessingStatus kvaccessor.KVAccessor

	// MaxRetries of the task processing
	MaxRetries int
}

func (opts *Options) validate() error {
	if opts.Driver == nil {
		return ErrStorageCollectionIsRequred
	}
	return nil
}

// Option contains basic storage properties
type Option func(opts *Options)

// WithDatabase connector interface
func WithDatabase(database DB) Option {
	return func(opts *Options) {
		opts.Database = database
	}
}

// WithDriver object accessor interface
func WithDriver(driver npio.StorageAccessor) Option {
	return func(opts *Options) {
		opts.Driver = driver
	}
}

// WithProcessingStatus object status accessor interface
func WithProcessingStatus(processingStatus kvaccessor.KVAccessor) Option {
	return func(opts *Options) {
		opts.ProcessingStatus = processingStatus
	}
}

// WithConverters list of processors
func WithConverters(converters ...converters.Converter) Option {
	return func(opts *Options) {
		opts.Converters = append(opts.Converters, converters...)
	}
}

// WithMaxRetries of the task execution
func WithMaxRetries(maxRetries int) Option {
	return func(opts *Options) {
		opts.MaxRetries = maxRetries
	}
}
