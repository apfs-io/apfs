package storage

import (
	"net/url"

	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	storio "github.com/apfs-io/apfs/internal/storio"
)

type uploadOption struct {
	customID      storio.ObjectID
	tags          []string
	params        map[string][]string
	overwrite     bool
	contentLength int64  // for validation
	contentType   string // for validation
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

// WithContentLength sets the expected file size for validation.
func WithContentLength(size int64) UploadOption {
	return func(opt *uploadOption) {
		opt.contentLength = size
	}
}

// WithContentType sets the declared MIME type for validation.
func WithContentType(ct string) UploadOption {
	return func(opt *uploadOption) {
		opt.contentType = ct
	}
}

// WithCustomID sets the custom object ID for the upload.
func WithCustomID(id storio.ObjectID) UploadOption {
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
	Driver storio.StorageAccessor

	// Processing status KeyValue accessor.
	// contains statuses of the object processing stages
	processingStatus kvaccessor.KVAccessor
}

func (opts *Options) validate() error {
	if opts.Driver == nil {
		return ErrStorageCollectionIsRequired
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
func WithDriver(driver storio.StorageAccessor) Option {
	return func(opts *Options) {
		opts.Driver = driver
	}
}

// WithProcessingStatus object status accessor interface
func WithProcessingStatus(processingStatus kvaccessor.KVAccessor) Option {
	return func(opts *Options) {
		opts.processingStatus = processingStatus
	}
}
