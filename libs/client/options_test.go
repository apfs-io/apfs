package client

import "testing"

func TestOptions(t *testing.T) {
	t.Run("WithGroup", func(t *testing.T) {
		t.Run("UploadOptions", func(t *testing.T) {
			o := &RequestOptions{}
			opts := []RequestOption{
				WithGroup("test"),
			}
			opts[0](o)
			if o.group != "test" {
				t.Errorf("expected group to be 'test', got %q", o.group)
			}
		})
	})
	t.Run("WithGRPCCallOpts", func(t *testing.T) {
		t.Run("UploadOptions", func(t *testing.T) {
			o := &RequestOptions{}
			opts := []RequestOption{
				WithGRPCCallOpts(),
			}
			opts[0](o)
			if len(o.grpcOpts) != 0 {
				t.Errorf("expected grpcOpts to be empty, got %v", o.grpcOpts)
			}
		})
	})
	t.Run("WithCustomID", func(t *testing.T) {
		o := &RequestOptions{}
		WithCustomID("test")(o)
		if o.customID != "test" {
			t.Errorf("expected customID to be 'test', got %q", o.customID)
		}
	})
	t.Run("WithTags", func(t *testing.T) {
		o := &RequestOptions{}
		WithTags("tag1", "tag2")(o)
		if len(o.tags) != 2 {
			t.Errorf("expected tags to have 2 elements, got %v", o.tags)
		}
	})
}
