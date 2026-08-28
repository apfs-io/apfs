package v1

import (
	"context"
	"io"

	"github.com/apfs-io/apfs/internal/preview"
	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/libs/storerrors"
)

type objectPreview struct {
	ContentType string
	Body        []byte
	Reader      io.ReadCloser
}

func (p *objectPreview) Close() {
	if p != nil && p.Reader != nil {
		_ = p.Reader.Close()
	}
}

func (s *server) resolvePreview(ctx context.Context, objectID string) (*objectPreview, error) {
	sObject, err := s.store.Object(ctx, objectID)
	if err != nil {
		return nil, err
	}
	if sObject == nil {
		return nil, storerrors.WrapNotFound(objectID, nil)
	}
	return s.previewForObject(ctx, sObject)
}

func (s *server) previewForObject(ctx context.Context, obj storio.Object) (*objectPreview, error) {
	ct := ""
	if meta := obj.MetaOrNew(); meta != nil {
		ct = meta.Main.ContentType
	}
	if preview.IsImage(ct) {
		_, data, err := s.store.OpenObject(ctx, obj, "")
		if err != nil {
			return nil, err
		}
		return &objectPreview{ContentType: ct, Reader: data}, nil
	}
	iconType, body := preview.IconForContentType(ct)
	return &objectPreview{ContentType: iconType, Body: body}, nil
}

// Preview returns display bytes and mime type for the object's main file.
func (s *server) Preview(ctx context.Context, obj *protocol.ObjectID) (*protocol.PreviewResponse, error) {
	p, err := s.resolvePreview(ctx, obj.GetId())
	if err != nil {
		if storerrors.IsNotFound(err) {
			return &protocol.PreviewResponse{
				Status:  protocol.ResponseStatusCode_NOT_FOUND,
				Message: err.Error(),
			}, nil
		}
		return &protocol.PreviewResponse{
			Status:  protocol.ResponseStatusCode_FAILED,
			Message: err.Error(),
		}, nil
	}
	defer p.Close()

	body := p.Body
	if p.Reader != nil {
		body, err = io.ReadAll(p.Reader)
		if err != nil {
			return &protocol.PreviewResponse{
				Status:  protocol.ResponseStatusCode_FAILED,
				Message: err.Error(),
			}, nil
		}
	}
	return &protocol.PreviewResponse{
		Status:      protocol.ResponseStatusCode_OK,
		Message:     "ok",
		ContentType: p.ContentType,
		Content:     body,
	}, nil
}
