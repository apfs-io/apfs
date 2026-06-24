package processor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor/memory"
	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

// Error definitions for specific failure scenarios
var (
	ErrStorageNoConverters       = errors.New("[processor] no converter registered")
	ErrStorageObjectInProcessing = errors.New("[processor] object in processing")
	ErrStorageInvalidAction      = errors.New("[processor] invalid action")
)

// Processor orchestrates manifest-driven task processing.
// It operates on models.Manifest (v1) directly; once Phase 4 replaces it
// with the Workflow executor this struct will be removed.
type Processor struct {
	mx sync.Mutex

	storage Storage

	driver storio.StorageAccessor

	converters []converters.Converter

	processingStatus kvaccessor.KVAccessor

	maxRetries int
}

// NewProcessor creates a Processor with the provided options.
func NewProcessor(opts ...Option) (*Processor, error) {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}
	if err := options.validate(); err != nil {
		return nil, err
	}
	if options.processingStatus == nil {
		options.processingStatus = memory.NewKVMemory(time.Minute * 20)
	}
	return &Processor{
		storage:          options.storage,
		driver:           options.driver,
		converters:       options.converters,
		processingStatus: options.processingStatus,
		maxRetries:       max(options.maxRetries, 1),
	}, nil
}

// ProcessTasks processes pending manifest tasks for an object.
// It returns (completed bool, error).
func (s *Processor) ProcessTasks(ctx context.Context, obj any, maxTasks, maxStages int) (bool, error) {
	cObject, err := s.storage.Object(ctx, obj)
	if err != nil {
		return false, err
	}

	manifest := workflowToManifest(s.storage.ObjectWorkflow(ctx, cObject))

	if manifest.TaskCount() > 0 && len(s.converters) == 0 {
		return false, ErrStorageNoConverters
	}

	if !s.startProcessing(ctx, cObject) {
		return false, errors.Wrap(ErrStorageObjectInProcessing, cObject.ID().String())
	}

	defer func() {
		if upErr := s.storage.UpdateObjectInfo(ctx, cObject); upErr != nil {
			ctxlogger.Get(ctx).Error("update meta info",
				zap.String(`object_bucket`, cObject.Bucket()),
				zap.String(`object_path`, cObject.Path()),
				zap.Error(upErr))
			if err == nil {
				err = upErr
			}
		}
		if s.getProcessingStatus(ctx, cObject).IsProcessing() {
			if sErr := s.setProcessingStatus(ctx, cObject, processingStatusBy(cObject, manifest, err)); sErr != nil {
				ctxlogger.Get(ctx).Error(`update processing status`,
					zap.String(`object_bucket`, cObject.Bucket()),
					zap.String(`object_path`, cObject.Path()), zap.Error(sErr))
			}
		}
	}()

	meta := cObject.MetaOrNew()
	meta.ManifestVersion = manifest.GetVersion()

PROCESSING_LOOP:
	for _, stage := range manifest.GetStages() {
		for _, task := range stage.Tasks {
			// Skip if this target already exists in meta
			if task.Target != "" && meta.ItemByName(task.Target) != nil {
				continue
			}
			if !s.canProcess(task) {
				ctxlogger.Get(ctx).Debug(`skip task`,
					zap.String(`task_id`, task.ID),
					zap.String(`object_bucket`, cObject.Bucket()),
					zap.String(`object_path`, cObject.Path()),
					zap.String("reason", "no suitable converter"))
				continue
			}
			if err := s.executeTask(ctx, cObject, manifest, task); err != nil {
				ctxlogger.Get(ctx).Error("execute task",
					zap.String(`task_id`, task.ID),
					zap.String(`object_bucket`, cObject.Bucket()),
					zap.String(`object_path`, cObject.Path()),
					zap.Error(err))
				return false, err
			}
			if maxTasks--; maxTasks == 0 {
				break PROCESSING_LOOP
			}
		}
		if maxStages--; maxStages == 0 {
			break
		}
	}

	// All manifest targets produced?
	return s.isComplete(cObject, manifest), err
}

// isComplete returns true when all REQUIRED manifest targets exist in meta.Items.
// Tasks with Required=false (optional/can-fail tasks) are ignored.
func (s *Processor) isComplete(cObject storio.Object, manifest *models.Manifest) bool {
	if manifest == nil || manifest.TaskCount() == 0 {
		return true
	}
	meta := cObject.MetaOrNew()
	for _, stage := range manifest.GetStages() {
		for _, task := range stage.Tasks {
			if task.Target == "" {
				continue
			}
			// Required tasks must produce their target file.
			// !Required means the task is optional (inverted semantics).
			if task.Required && meta.ItemByName(task.Target) == nil {
				return false
			}
		}
	}
	return true
}

// executeTask runs a single manifest task on cObject.
func (s *Processor) executeTask(ctx context.Context, cObject storio.Object, manifest *models.Manifest, task *models.ManifestTask) error {
	var (
		err            error
		finalizers     []io.Closer
		meta           = cObject.MetaOrNew()
		sourceMeta     = meta.ItemByName(task.Source)
		targetFilename string
		targetMeta     *models.ItemMeta
		inputStream    io.Reader
		out            converters.Output
	)

	if sourceMeta == nil {
		return fmt.Errorf("source meta information not found [%s]", task.Source)
	}

	if task.Target == "" {
		targetMeta = sourceMeta
	} else {
		targetMeta = meta.ItemByName(task.Target)
		if targetMeta == nil {
			targetMeta = &models.ItemMeta{}
		}
		targetFilename = models.SourceFilename(
			task.Target,
			defStr(filepath.Ext(task.Target),
				defStr(sourceMeta.ObjectTypeExt(), meta.Main.NameExt)),
		)
	}

	ctxlogger.Get(ctx).Debug("execute processing task",
		zap.String(`task_id`, task.ID),
		zap.String(`task_source`, task.Source),
		zap.String(`task_target`, task.Target),
		zap.String(`target_objectname`, targetFilename),
		zap.Int(`stage_count`, len(manifest.Stages)),
		zap.Int(`task_count`, manifest.TaskCount()),
		zap.Int(`action_count`, len(task.Actions)))

	defer func() {
		for _, obj := range finalizers {
			if err := obj.Close(); err != nil {
				ctxlogger.Get(ctx).Error(`close processing object`, zap.Error(err))
			}
		}
	}()

	if obj, err := s.driver.Read(ctx, cObject, task.Source); err != nil {
		return err
	} else {
		finalizers = append(finalizers, obj)
		inputStream = obj
	}

	for _, action := range task.Actions {
		conv := s.converterByAction(action)
		if conv == nil {
			err = errors.Wrap(ErrStorageInvalidAction, action.Name)
			break
		}
		in := converters.NewInput(inputStream, task, action, sourceMeta)
		out = converters.NewOutput(targetMeta)

		if err = conv.Process(in, out); err != nil && err != converters.ErrSkip {
			err = errors.Wrap(err, "process action")
			break
		}

		if err == converters.ErrSkip || out.ObjectReader() == nil {
			inputStream, err = resetReader(in.ObjectReader())
			if err != nil && !errors.Is(err, errReaderResetPosition) {
				err = errors.Wrap(err, "reset object")
				break
			}
			err = nil
			continue
		}

		if !out.IsEqual(in) {
			if cl := getFinishCloser(conv, in, out); cl != nil {
				finalizers = append(finalizers, cl)
			}
		}
		inputStream = out.ObjectReader()

		if err != nil && !errors.Is(err, errReaderResetPosition) {
			break
		}
	}

	if targetFilename == "" {
		targetFilename = models.SourceFilename(task.Source, meta.Main.NameExt)
		out = nil
	}

	// On error: if the task is not required, swallow the error
	if err != nil && !task.Required {
		ctxlogger.Get(ctx).Warn("task failed but not required",
			zap.String("task_id", task.ID), zap.Error(err))
		err = nil
	}

	// Always persist metadata for tasks with a named target (even failed optional ones),
	// so ExcessItemsFromManifest can discover and clean them up later if needed.
	if err == nil && (task.Target != "" || out != nil && !isEmptyOutput(out.ObjectReader()) || !targetMeta.IsEmpty()) {
		targetMeta.UpdatedAt = time.Now()
		targetMeta.UpdateName(targetFilename)

		updateProcessingState(cObject, manifest)

		var outputStream io.Reader
		if out != nil {
			outputStream = out.ObjectReader()
		}

		if isNil(outputStream) {
			if upErr := s.driver.UpdateMeta(ctx, cObject, targetFilename, targetMeta); upErr != nil {
				return errors.Wrapf(upErr, "execute task update meta:'%s'", targetFilename)
			}
		} else {
			if upErr := s.driver.Update(ctx, cObject, targetFilename, outputStream, targetMeta); upErr != nil {
				return errors.Wrapf(upErr, "execute task upload:'%s'", targetFilename)
			}
		}

		// Store artifact in meta
		if task.Target != "" {
			targetMeta.Role = task.ID
			meta.SetItem(targetMeta)
		}
	}

	if err != nil && task.Required {
		return err
	}
	return nil
}

func (s *Processor) converterByAction(action *models.Action) converters.Converter {
	for _, conv := range s.converters {
		if conv.Test(action) {
			return conv
		}
	}
	return nil
}

func (s *Processor) canProcess(task *models.ManifestTask) bool {
	for _, action := range task.Actions {
		if s.converterByAction(action) != nil {
			return true
		}
	}
	return len(task.Actions) == 0
}

func (s *Processor) startProcessing(ctx context.Context, cObject storio.Object) bool {
	s.mx.Lock()
	defer s.mx.Unlock()

	key := processingKey(cObject)
	status, _ := s.processingStatus.Get(ctx, key)

	if models.ObjectStatus(status).IsEmpty() {
		return s.processingStatus.TrySet(ctx, key, models.StatusProcessing.String()) == nil
	}
	return false
}

func (s *Processor) getProcessingStatus(ctx context.Context, cObject storio.Object) models.ObjectStatus {
	s.mx.Lock()
	defer s.mx.Unlock()
	return GetProcessingStatus(ctx, s.processingStatus, s.storage, cObject)
}

func (s *Processor) setProcessingStatus(ctx context.Context, cObject storio.Object, status models.ObjectStatus) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	return SetProcessingStatus(ctx, s.processingStatus, cObject, status)
}

func getFinishCloser(conv converters.Converter, in converters.Input, out converters.Output) io.Closer {
	if finisher, ok := conv.(converters.Finisher); ok {
		return closer(func() error { return finisher.Finish(in, out) })
	}
	if out != nil {
		if c, ok := out.ObjectReader().(io.Closer); ok {
			return c
		}
	}
	return nil
}

func isEmptyOutput(out io.Reader) bool {
	switch v := out.(type) {
	case nil:
		return true
	case *bytes.Buffer:
		return v.Len() == 0
	case *os.File:
		return v.Name() == ""
	}
	return false
}
