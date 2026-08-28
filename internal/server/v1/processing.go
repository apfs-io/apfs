package v1

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	"github.com/apfs-io/apfs/internal/context/ctxstatusstream"
	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/libs/storerrors"
	"github.com/apfs-io/apfs/models"
)

func (s *server) withStatusStream(ctx context.Context) context.Context {
	if s.statusStream != nil {
		return ctxstatusstream.WithPublisher(ctx, s.statusStream)
	}
	return ctx
}

func (s *server) loadProcessingState(ctx context.Context, obj storio.Object) *models.ProcessingState {
	state, _ := s.store.GetProcessingState(ctx, obj.ID().String())
	if state == nil {
		wf := s.store.ObjectWorkflow(ctx, obj)
		state = objectStatusToProcessingState(obj, wf)
	}
	return state
}

// GetProcessingState returns the current processing state (no side effects).
func (s *server) GetProcessingState(ctx context.Context, obj *protocol.ObjectID) (*protocol.ProcessingStateResponse, error) {
	sObject, err := s.store.Object(ctx, obj.GetId())
	if err != nil && !storerrors.IsNotFound(err) {
		return &protocol.ProcessingStateResponse{
			Status:  protocol.ResponseStatusCode_FAILED,
			Message: err.Error(),
		}, nil
	}
	if sObject == nil || storerrors.IsNotFound(err) {
		return &protocol.ProcessingStateResponse{
			Status:  protocol.ResponseStatusCode_NOT_FOUND,
			Message: "Not found",
		}, nil
	}
	state := s.loadProcessingState(ctx, sObject)
	return &protocol.ProcessingStateResponse{
		Status:  protocol.ResponseStatusCode_OK,
		Message: "ok",
		State:   protocol.ProcessingStateFromModel(state, true),
	}, nil
}

// EnsureProcessing republishes status to the stream and resumes processing
// when the object is not yet terminal.
func (s *server) EnsureProcessing(ctx context.Context, obj *protocol.ObjectID) (*protocol.ProcessingStateResponse, error) {
	ctx = s.withStatusStream(ctx)
	state, err := s.ensureProcessing(ctx, obj.GetId())
	if err != nil {
		if storerrors.IsNotFound(err) {
			return &protocol.ProcessingStateResponse{
				Status:  protocol.ResponseStatusCode_NOT_FOUND,
				Message: err.Error(),
			}, nil
		}
		return &protocol.ProcessingStateResponse{
			Status:  protocol.ResponseStatusCode_FAILED,
			Message: err.Error(),
		}, nil
	}
	return &protocol.ProcessingStateResponse{
		Status:  protocol.ResponseStatusCode_OK,
		Message: "ok",
		State:   protocol.ProcessingStateFromModel(state, true),
	}, nil
}

func (s *server) ensureProcessing(ctx context.Context, objectID string) (*models.ProcessingState, error) {
	sObject, err := s.store.Object(ctx, objectID)
	if err != nil {
		return nil, err
	}
	if sObject == nil {
		return nil, storerrors.WrapNotFound(objectID, nil)
	}

	state := s.loadProcessingState(ctx, sObject)
	_ = ctxstatusstream.PublishFromState(ctx, state, state.Status.IsTerminal())

	if state.Status.IsTerminal() {
		if s.inflight != nil {
			_ = s.inflight.Remove(ctx, objectID)
		}
		return state, nil
	}

	if s.inflight != nil {
		_ = s.inflight.Add(ctx, objectID)
		if s.inflight.IsBusy(objectID) {
			return state, nil
		}
	}
	s.updateObjectState(ctx, objectID)
	return state, nil
}

// StartStallWatchdog periodically re-checks inflight objects older than interval.
// A zero interval disables the watchdog.
func (s *server) StartStallWatchdog(ctx context.Context, interval time.Duration) {
	if interval <= 0 || s.inflight == nil {
		return
	}
	go s.runStallWatchdog(ctx, interval)
}

func (s *server) runStallWatchdog(ctx context.Context, interval time.Duration) {
	logger := ctxlogger.Get(ctx)
	logger.Info("starting processing stall watchdog", zap.Duration("interval", interval))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tickStallWatchdog(ctx, interval)
		}
	}
}

func (s *server) tickStallWatchdog(ctx context.Context, interval time.Duration) {
	ctx = s.withStatusStream(ctx)
	ids, err := s.inflight.ListOlderThan(ctx, interval)
	if err != nil {
		ctxlogger.Get(ctx).Error("stall watchdog list inflight", zap.Error(err))
		return
	}
	for _, id := range ids {
		if _, err := s.ensureProcessing(ctx, id); err != nil && !storerrors.IsNotFound(err) {
			ctxlogger.Get(ctx).Warn("stall watchdog ensure processing",
				zap.String("object_id", id), zap.Error(err))
		}
	}
}
