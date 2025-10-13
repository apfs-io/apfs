package client

import "google.golang.org/grpc"

// RequestOptions is a type for upload options.
type RequestOptions struct {
	customID  string
	group     string
	grpcOpts  []grpc.CallOption
	tags      []string
	overwrite bool
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
