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
	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor/memory"
	"github.com/apfs-io/apfs/models"
)

// Error definitions for specific failure scenarios
var (
	ErrStorageNoConverters       = errors.New("[processor] no eny converter registered") // No converters available for processing
	ErrStorageObjectInProcessing = errors.New("[processor] object in processing")        // Object is already being processed
	ErrStorageInvalidAction      = errors.New("[processor] invalid action")              // Invalid action encountered
)

// Processor struct orchestrates task processing
type Processor struct {
	// Mutex for thread-safe processing status updates
	mx sync.Mutex

	// Storage interface for object operations
	storage Storage

	// Accessor for file objects
	driver npio.StorageAccessor

	// List of converters for task processing
	converters []converters.Converter

	// Key-value accessor for processing statuses
	processingStatus kvaccessor.KVAccessor

	// Maximum number of retries for task processing
	maxRetries int
}

// NewProcessor creates a new Processor instance with the provided options
func NewProcessor(opts ...Option) (*Processor, error) {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}
	// Validate options and set default processing status if not provided
	if err := options.validate(); err != nil {
		return nil, err
	}
	if options.processingStatus == nil {
		options.processingStatus = memory.NewKVMemory(time.Minute * 20) // Default in-memory KV store
	}
	return &Processor{
		storage:          options.storage,
		driver:           options.driver,
		converters:       options.converters,
		processingStatus: options.processingStatus,
		maxRetries:       max(options.maxRetries, 1), // Ensure at least one retry
	}, nil
}

// ProcessTasks processes tasks for a given object and returns completion status and error
func (s *Processor) ProcessTasks(ctx context.Context, obj any, maxTasks, maxStages int) (bool, error) {
	cObject, err := s.storage.Object(ctx, obj) // Open the object
	if err != nil {
		return false, err
	}

	var (
		meta     = cObject.MustMeta()                     // Retrieve metadata of the object
		manifest = s.storage.ObjectManifest(ctx, cObject) // Retrieve the object's manifest
	)

	// Ensure there are converters available if tasks exist
	if manifest.TaskCount() > 0 && len(s.converters) < 1 {
		return false, ErrStorageNoConverters
	}

	// Check if the object is already being processed
	if !s.startProcessing(ctx, cObject) {
		return false, errors.Wrap(ErrStorageObjectInProcessing, cObject.ID().String())
	}

	// Defer cleanup and status updates
	defer func() {
		// Update object metadata
		if upErr := s.storage.UpdateObjectInfo(ctx, cObject); upErr != nil {
			ctxlogger.Get(ctx).Error("update meta info",
				zap.String(`object_bucket`, cObject.Bucket()),
				zap.String(`object_path`, cObject.Path()),
				zap.Error(upErr))
			if err == nil {
				err = upErr
			}
		}
		// Update processing status if still marked as processing
		if s.getProcessingStatus(ctx, cObject).IsProcessing() {
			if sErr := s.setProcessingStatus(ctx, cObject, processingStatusBy(cObject, manifest, err)); sErr != nil {
				ctxlogger.Get(ctx).Error(`update processing status`,
					zap.String(`object_bucket`, cObject.Bucket()),
					zap.String(`object_path`, cObject.Path()), zap.Error(sErr))
			}
		}
	}()

	// Update manifest version in metadata
	meta.ManifestVersion = manifest.GetVersion()

	// Begin task processing loop
PROCESSING_LOOP:
	for _, stage := range manifest.GetStages() {
		// Iterate through tasks in the stage
		for _, task := range stage.Tasks {
			// Skip tasks that are already complete
			if meta.IsProcessingCompleteTask(task, s.maxRetries) {
				continue
			}
			// Skip tasks if no suitable converter is available
			if !s.canProcess(task) {
				ctxlogger.Get(ctx).Debug(`skip task`,
					zap.String(`task_id`, task.ID),
					zap.Int(`task_actions`, len(task.Actions)),
					zap.String(`object_bucket`, cObject.Bucket()),
					zap.String(`object_path`, cObject.Path()),
					zap.Stack("stack"),
					zap.String("reason", "no suitable converter"))
				break // Skip the stage
			}
			// Execute the task
			if err := s.executeTask(ctx, cObject, manifest, task); err != nil {
				ctxlogger.Get(ctx).Error("execute task",
					zap.String(`task_id`, task.ID),
					zap.Int(`task_actions`, len(task.Actions)),
					zap.String(`object_bucket`, cObject.Bucket()),
					zap.String(`object_path`, cObject.Path()),
					zap.Error(err),
					zap.Stack("stack"))
				return false, err
			}
			// Stop processing if maxTasks limit is reached
			if maxTasks--; maxTasks == 0 {
				break PROCESSING_LOOP
			}
		}
		// Stop processing if maxStages limit is reached
		if maxStages--; maxStages == 0 {
			break
		}
	}

	// Return true if manifest is nil or all tasks are complete
	return manifest == nil || meta.IsProcessingComplete(manifest), err
}

// executeTask processes an individual task
func (s *Processor) executeTask(ctx context.Context, cObject npio.Object, manifest *models.Manifest, task *models.ManifestTask) error {
	var (
		err            error
		finalizers     []io.Closer // List of resources to close after processing
		meta           = cObject.MustMeta()
		sourceMeta     = meta.ItemByName(task.Source) // Metadata of the source item
		targetFilename string
		targetMeta     *models.ItemMeta
		inputStream    io.Reader
		out            converters.Output
	)

	// Ensure source metadata exists
	if sourceMeta == nil {
		return fmt.Errorf("source meta information not found [%s]", task.Source)
	}

	// Determine target metadata
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

	// Log task execution details
	ctxlogger.Get(ctx).Debug("execute processing task",
		zap.String(`task_id`, task.ID),
		zap.String(`task_source`, task.Source),
		zap.String(`task_target`, task.Target),
		zap.String(`target_objectname`, targetFilename),
		zap.Int(`stage_count`, len(manifest.Stages)),
		zap.Int(`task_count`, manifest.TaskCount()),
		zap.Int(`action_count`, len(task.Actions)))

	// Defer cleanup of IO resources
	defer func() {
		for _, obj := range finalizers {
			if err := obj.Close(); err != nil {
				ctxlogger.Get(ctx).Error(`close processing object`, zap.Error(err))
			}
		}
	}()

	// Open source stream
	if obj, err := s.driver.Read(ctx, cObject, task.Source); err != nil {
		return err
	} else {
		finalizers = append(finalizers, obj)
		inputStream = obj
	}

	// Process each action in the task
	for _, action := range task.Actions {
		conv := s.converterByAction(action) // Retrieve the appropriate converter
		if conv == nil {
			err = errors.Wrap(ErrStorageInvalidAction, action.Name)
			break
		}
		in := converters.NewInput(inputStream, task, action, sourceMeta)
		out = converters.NewOutput(targetMeta)

		// Execute the action using the converter
		if err = conv.Process(in, out); err != nil && err != converters.ErrSkip {
			err = errors.Wrap(err, "process action")
			break
		}

		// Handle skipped actions or reset input stream
		if err == converters.ErrSkip || out.ObjectReader() == nil {
			inputStream, err = resetReader(in.ObjectReader())
			if err != nil && !errors.Is(err, errReaderResetPosition) {
				err = errors.Wrap(err, "reset object")
				break
			}
			err = nil
			continue
		}

		// Finalize processing if output differs from input
		if !out.IsEqual(in) {
			if cl := getFinishCloser(conv, in, out); cl != nil {
				finalizers = append(finalizers, cl)
			}
		}

		// Update input stream for the next action
		inputStream = out.ObjectReader()

		if err != nil && !errors.Is(err, errReaderResetPosition) {
			break
		}
	}

	// Update metadata and save the target file
	if targetFilename == "" {
		targetFilename = models.SourceFilename(task.Source, meta.Main.NameExt)
		out = nil
	}

	// Mark the task as complete or failed if an error occurred
	meta.Complete(targetMeta, task, err)

	if err != nil || (out != nil && !isEmptyOutput(out.ObjectReader())) || !targetMeta.IsEmpty() {
		targetMeta.UpdatedAt = time.Now()
		targetMeta.UpdateName(targetFilename)
		var outputStream io.Reader
		if out != nil {
			outputStream = out.ObjectReader()
		}

		// Update the processing status
		updateProcessingState(cObject, manifest)

		// Upload the processed output
		if isNil(outputStream) {
			if err := s.driver.UpdateMeta(ctx, cObject, targetFilename, targetMeta); err != nil {
				return errors.Wrapf(err, "execute task update meta:'%s'", targetFilename)
			}
		} else {
			if err := s.driver.Update(ctx, cObject, targetFilename, outputStream, targetMeta); err != nil {
				return errors.Wrapf(err, "execute task upload:'%s'", targetFilename)
			}
		}
	}

	// If the task is not required, then skip error handling
	if err != nil && task.Required {
		return err
	}

	return nil
}

// Additional helper methods
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

func (s *Processor) startProcessing(ctx context.Context, cObject npio.Object) bool {
	s.mx.Lock()
	defer s.mx.Unlock()

	key := processingKey(cObject)
	status, _ := s.processingStatus.Get(ctx, key)

	if models.ObjectStatus(status).IsEmpty() {
		return s.processingStatus.TrySet(ctx, key, status) == nil
	}
	return false
}

func (s *Processor) getProcessingStatus(ctx context.Context, cObject npio.Object) models.ObjectStatus {
	s.mx.Lock()
	defer s.mx.Unlock()
	return GetProcessingStatus(ctx, s.processingStatus, s.storage, cObject)
}

func (s *Processor) setProcessingStatus(ctx context.Context, cObject npio.Object, status models.ObjectStatus) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	return SetProcessingStatus(ctx, s.processingStatus, cObject, status)
}

func getFinishCloser(conv converters.Converter, in converters.Input, out converters.Output) io.Closer {
	if finisher, ok := conv.(converters.Finisher); ok {
		return closer(func() error { return finisher.Finish(in, out) })
	}
	if out != nil {
		if closer, ok := out.ObjectReader().(io.Closer); ok {
			return closer
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
