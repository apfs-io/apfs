package client

import "google.golang.org/grpc"

// RequestOptions is a type for upload options.
type RequestOptions struct {
	customID  string
	group     string
	grpcOpts  []grpc.CallOption
	tags      []string
	overwrite bool

	// Optional data to include in the object response (single-request fetch).
	includeWorkflow  bool // include bucket workflow manifest
	includeState     bool // include processing state (counters only)
	includeStateFull bool // include full job details (requires includeState=true)
}

func (o *RequestOptions) prepareGroup(defaultGroup string) {
	if o.group == "" {
		o.group = defaultGroup
	}
}

// RequestOption is a type for a function that modifies RequestOptions.
type RequestOption func(*RequestOptions)

// WithGroup sets the group option for the upload operation.
func WithGroup(group string) func(o *RequestOptions) {
	return func(o *RequestOptions) {
		o.group = group
	}
}

// WithGRPCCallOpts sets the gRPC call options for the upload operation.
func WithGRPCCallOpts(opts ...grpc.CallOption) func(o *RequestOptions) {
	return func(o *RequestOptions) {
		o.grpcOpts = opts
	}
}

// WithCustomID sets the custom ID option for the upload operation.
func WithCustomID(id string) func(*RequestOptions) {
	return func(o *RequestOptions) {
		o.customID = id
	}
}

// WithTags sets the tags option for the upload operation.
func WithTags(opts ...string) func(*RequestOptions) {
	return func(o *RequestOptions) {
		o.tags = opts
	}
}

// WithOverwrite sets the overwrite option for the upload operation.
func WithOverwrite(opts ...bool) func(*RequestOptions) {
	return func(o *RequestOptions) {
		if len(opts) > 0 {
			o.overwrite = opts[0]
		} else {
			o.overwrite = true
		}
	}
}

// WithWorkflow instructs the server to include the bucket workflow manifest in the response.
func WithWorkflow() RequestOption {
	return func(o *RequestOptions) { o.includeWorkflow = true }
}

// WithState instructs the server to include a compact ProcessingState (counters + status only).
func WithState() RequestOption {
	return func(o *RequestOptions) { o.includeState = true }
}

// WithFullState instructs the server to include the full ProcessingState (counters + job details).
func WithFullState() RequestOption {
	return func(o *RequestOptions) {
		o.includeState = true
		o.includeStateFull = true
	}
}
