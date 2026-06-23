package client

import (
	"context"
	"io"

	"github.com/apfs-io/apfs/models"
)

// Group is a fluent scoped client bound to a single bucket/group.
// Obtain a Group via Client.Group(name).
//
//	group := cl.Group("videos")
//	obj, err := group.Upload(ctx, file, apfs.WithTags("promo"))
//	state, err := group.ProcessingState(ctx, "video-id")
type Group struct {
	client *client
	name   string
}

// Group returns a Group scoped to the named bucket.
func (c *client) Group(name string) *Group {
	return &Group{client: c, name: name}
}

// Head returns the meta information for the named object.
func (g *Group) Head(ctx context.Context, id string, opts ...RequestOption) (*models.Object, error) {
	all := append(opts, WithGroupOpt(g.name))
	return g.client.Head(ctx, &ObjectID{Id: id}, all...)
}

// Get returns the named object and its data stream.
func (g *Group) Get(ctx context.Context, id string, opts ...RequestOption) (*models.Object, io.ReadCloser, error) {
	all := append(opts, WithGroupOpt(g.name))
	return g.client.Get(ctx, &ObjectID{Id: id}, all...)
}

// Refresh triggers re-processing of the named object.
func (g *Group) Refresh(ctx context.Context, id string, opts ...RequestOption) error {
	all := append(opts, WithGroupOpt(g.name))
	return g.client.Refresh(ctx, &ObjectID{Id: id}, all...)
}

// Upload uploads data to the group and returns the resulting object.
func (g *Group) Upload(ctx context.Context, data io.Reader, opts ...RequestOption) (*models.Object, error) {
	all := append(opts, WithGroupOpt(g.name))
	return g.client.Upload(ctx, data, all...)
}

// UploadFile uploads a file from disk to the group.
func (g *Group) UploadFile(ctx context.Context, filepath string, opts ...RequestOption) (*models.Object, error) {
	all := append(opts, WithGroupOpt(g.name))
	return g.client.UploadFile(ctx, filepath, all...)
}

// Delete removes an object (or named subfiles) from the group.
func (g *Group) Delete(ctx context.Context, id string, names ...string) error {
	return g.client.Delete(ctx, &ObjectIDNames{Id: id, Names: names}, WithGroupOpt(g.name))
}

// SetWorkflow stores the workflow manifest for this group.
func (g *Group) SetWorkflow(ctx context.Context, w *models.Workflow, opts ...RequestOption) error {
	all := append(opts, WithGroupOpt(g.name))
	return g.client.SetWorkflow(ctx, w, all...)
}

// GetWorkflow reads the workflow manifest for this group.
func (g *Group) GetWorkflow(ctx context.Context, opts ...RequestOption) (*models.Workflow, error) {
	all := append(opts, WithGroupOpt(g.name))
	return g.client.GetWorkflow(ctx, all...)
}

// ProcessingState returns the current processing state for the given object ID.
func (g *Group) ProcessingState(ctx context.Context, id string, opts ...RequestOption) (*models.ProcessingState, error) {
	obj, err := g.Head(ctx, id, opts...)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	status := models.ProcessingStatusPending
	switch obj.Status {
	case models.StatusOK:
		status = models.ProcessingStatusCompleted
	case models.StatusProcessing:
		status = models.ProcessingStatusRunning
	case models.StatusError:
		status = models.ProcessingStatusFailed
	}
	return &models.ProcessingState{
		ObjectID: id,
		Status:   status,
	}, nil
}

// WatchProgress calls handler on each state update until the object reaches a
// terminal state or the context is cancelled.
func (g *Group) WatchProgress(ctx context.Context, id string, handler func(*models.ProcessingState)) error {
	if handler == nil {
		return nil
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}
	state, err := g.ProcessingState(ctx, id)
	if err != nil {
		return err
	}
	if state != nil {
		handler(state)
		if state.Status.IsTerminal() {
			return nil
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return nil // single iteration in polling mode
}

// --- Convenience upload options ---

// WithID sets a custom object ID for the upload.
func WithID(id string) RequestOption {
	return WithCustomID(id)
}

// WithGroupOpt sets the group on a RequestOption.
func WithGroupOpt(group string) RequestOption {
	return WithGroup(group)
}
