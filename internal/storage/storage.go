//
// @project apfs 2018 - 2020
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2020
//

package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/object"
	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	"github.com/apfs-io/apfs/models"
)

// Error list...
var (
	ErrStorageInvalidParameterType = errors.New("[storage] invalid parameter type")
	ErrStorageNoConverters         = errors.New("[storage] no eny converter registered")
	ErrStorageCollectionIsRequred  = errors.New("[storage] collection accessor is required")
	ErrStorageObjectInProcessing   = errors.New("[storage] object in processing")
	ErrStorageInvalidGroupName     = errors.New("[storage] invalid group name")
	ErrStorageInvalidAction        = errors.New("[storage] invalid action")
)

// AllTasks defines task processing count
const AllTasks = -1

// AllStages defines stage processing count
const AllStages = -1

// Storage basic object
type Storage struct {
	mx sync.Mutex

	// database storage accessor
	db DB

	// collection of file objects
	driver npio.StorageAccessor

	// converter list of processing
	converters []converters.Converter

	// Processing status KeyValue accessor.
	// contains statuses of the object processing stages
	processingStatus kvaccessor.KVAccessor

	// Max task processing attempts
	maxRetries int
}

// NewStorage object who controles file storage
func NewStorage(options ...Option) *Storage {
	var opts Options
	for _, opt := range options {
		opt(&opts)
	}
	if err := opts.validate(); err != nil {
		panic(err)
	}
	return &Storage{
		db:               opts.Database,
		driver:           opts.Driver,
		processingStatus: opts.ProcessingStatus,
		converters:       opts.Converters,
		maxRetries:       opts.MaxRetries,
	}
}

// SetManifest for a group
func (s *Storage) SetManifest(ctx context.Context, group string, manifest *models.Manifest) error {
	return s.driver.UpdateManifest(ctx, group, manifest)
}

// GetManifest for a group
func (s *Storage) GetManifest(ctx context.Context, group string) (*models.Manifest, error) {
	return s.driver.ReadManifest(ctx, group)
}

// UploadFile into storage
func (s *Storage) UploadFile(ctx context.Context, group string, sourceFilePath string, options ...UploadOption) (npio.Object, error) {
	file, err := os.Open(sourceFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return s.Upload(ctx, group, file, options...)
}

// Upload data into storage
func (s *Storage) Upload(ctx context.Context, group string, data io.Reader, options ...UploadOption) (obj npio.Object, err error) {
	if group == "" {
		return nil, ErrStorageInvalidGroupName
	}
	var option uploadOption
	for _, opt := range options {
		opt(&option)
	}
	// Create new object container
	if obj, err = s.driver.Create(ctx, group, option.customID, option.overwrite, option.Params()); err != nil {
		return nil, err
	}
	// Upload object data
	if err = s.driver.Update(ctx, obj, models.OriginalFilename, data, nil); err == nil {
		if err = s.UpdateObjectInfo(ctx, obj); err != nil {
			if err2 := s.Delete(ctx, obj); err2 != nil {
				err = fmt.Errorf("%s [%s]", err.Error(), err2.Error())
			}
		}
	}
	if err != nil {
		obj = nil
	}
	return obj, err
}

// Open storage file
func (s *Storage) Open(ctx context.Context, obj any) (nObject npio.Object, err error) {
	switch v := obj.(type) {
	case string:
		objectRecord, err := s.db.Get(v)
		if err != nil || objectRecord == nil {
			ctxlogger.Get(ctx).Debug("load object cache", zap.Error(err),
				zap.String("object_id", v),
				zap.Stack("stack"))
			if nObject, err = s.driver.Open(ctx, npio.ObjectIDType(v)); err != nil {
				return nil, err
			}
			// Refresh object cache
			if objectMode, err := object.ToModel(nObject); err == nil {
				if err = s.db.Set(objectMode); err != nil {
					ctxlogger.Get(ctx).Error("update object object cache", zap.Error(err),
						zap.String("object_id", v))
				}
			}
		} else {
			nObject = object.FromModel(objectRecord)
		}
	case npio.Object:
		nObject = v
	default:
		return nil, errors.Wrapf(ErrStorageInvalidParameterType, `%T`, obj)
	}
	nObject.StatusUpdate(s.getProcessingStatus(ctx, nObject))
	return nObject, nil
}

// OpenObject to read data
func (s *Storage) OpenObject(ctx context.Context, obj any, name string) (_ io.ReadCloser, err error) {
	var nObject npio.Object
	switch v := obj.(type) {
	case string:
		if nObject, err = s.driver.Open(ctx, npio.ObjectIDType(v)); err != nil {
			return nil, err
		}
	case npio.Object:
		nObject = v
	default:
		return nil, errors.Wrapf(ErrStorageInvalidParameterType, `%T`, obj)
	}
	nObject.StatusUpdate(s.getProcessingStatus(ctx, nObject))
	return s.driver.Read(ctx, nObject, name)
}

// Delete object completeley
func (s *Storage) Delete(ctx context.Context, obj any, names ...string) (err error) {
	var nObject npio.Object
	if nObject, err = s.Open(ctx, obj); err != nil {
		if os.IsNotExist(err) {
			return s.db.Delete(objcID(obj))
		}
		return err
	}
	if err = s.driver.Remove(ctx, nObject, names...); err != nil {
		return err
	}
	return s.db.Delete(nObject.ID().String())
}

// Update information about object in database
func (s *Storage) UpdateObjectInfo(ctx context.Context, obj npio.Object) error {
	mObj, err := object.ToModel(obj)
	if err != nil {
		return err
	}
	return s.db.Set(mObj)
}

// ProcessTasks object removes all subobjects from current object
// and starts processing from of the file again
// Returns completion status and error
func (s *Storage) ProcessTasks(ctx context.Context, obj any, maxTasks, maxStages int) (bool, error) {
	cObject, err := s.Open(ctx, obj)
	if err != nil {
		return false, err
	}

	var (
		meta     = cObject.MustMeta()
		manifest = s.ObjectManifest(ctx, cObject)
	)

	if manifest.TaskCount() > 0 && len(s.converters) < 1 {
		return false, ErrStorageNoConverters
	}

	// Check if the object in processing status
	if !s.startProcessing(ctx, cObject) {
		return false, errors.Wrap(ErrStorageObjectInProcessing, cObject.ID().String())
	}

	defer func() {
		if upErr := s.UpdateObjectInfo(ctx, cObject); upErr != nil {
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

	//lint:ignore staticcheck Update manifest safely
	meta.ManifestVersion = manifest.GetVersion()

	// Begin task processing loop...
PROCESSING_LOOP:
	for _, stage := range manifest.GetStages() {
		// TODO: add stage processing lock
		for _, task := range stage.Tasks {
			if meta.IsProcessingCompleteTask(task, s.maxRetries) {
				continue
			}
			if !s.canProcess(task) {
				ctxlogger.Get(ctx).Debug(`skip task`,
					zap.String(`task_id`, task.ID),
					zap.Int(`task_actions`, len(task.Actions)),
					zap.String(`object_bucket`, cObject.Bucket()),
					zap.String(`object_path`, cObject.Path()))
				break // Skip the stage as converter is no active
			}
			// Save main status...
			if err := s.executeTask(ctx, cObject, manifest, task); err != nil {
				ctxlogger.Get(ctx).Error("execute task",
					zap.String(`task_id`, task.ID),
					zap.Int(`task_actions`, len(task.Actions)),
					zap.String(`object_bucket`, cObject.Bucket()),
					zap.String(`object_path`, cObject.Path()),
					zap.Error(err))
				return false, err
			}
			// if <= 0 then all tasks must be processed
			if maxTasks--; maxTasks == 0 {
				break PROCESSING_LOOP
			}
		}
		if maxStages--; maxStages == 0 {
			break
		}
	}
	// TODO: temporary logic ingection to avoid infinity looping {manifest cant be nil}
	return manifest == nil || meta.IsProcessingComplete(manifest), err
}

func (s *Storage) executeTask(ctx context.Context, cObject npio.Object, manifest *models.Manifest, task *models.ManifestTask) error {
	var (
		err            error
		finalizers     []io.Closer
		meta           = cObject.MustMeta()
		sourceMeta     = meta.ItemByName(task.Source)
		targetFilename string
		targetMeta     *models.ItemMeta
		inputStream    io.Reader
		out            converters.Output
	)

	if sourceMeta == nil {
		return fmt.Errorf("source meta information not found [%s]", task.Source)
	}

	// Reset target if not defined
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

	// Post-process all IO objects...
	// All open objects have to be closed
	defer func() {
		for _, obj := range finalizers {
			if err := obj.Close(); err != nil {
				ctxlogger.Get(ctx).Error(`close processing object`, zap.Error(err))
			}
		}
	}()

	// Open source stream file...
	if true {
		var obj io.ReadCloser
		if obj, err = s.driver.Read(ctx, cObject, task.Source); err != nil {
			return err
		}
		finalizers = append(finalizers, obj)
		inputStream = obj
	}

	// Execute all processing tasks
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

		// If output is not the same like input
		if !out.IsEqual(in) {
			// If implemented finisher interface than need to finish file processing
			if cl := getFinishCloser(conv, in, out); cl != nil {
				finalizers = append(finalizers, cl)
			}
		}

		// Redeclare the input stream
		inputStream = out.ObjectReader()

		if err != nil && !errors.Is(err, errReaderResetPosition) {
			break
		}
	}

	// Update source meta information target
	if targetFilename == "" {
		// Take extension from target meta and remove the old one if thare are differ
		targetFilename = models.SourceFilename(task.Source, meta.Main.NameExt)
		out = nil
	}

	// Complete task state update
	meta.Complete(targetMeta, task, err)

	// Save target file...
	if err != nil || (out != nil && out.ObjectReader() != nil) || !targetMeta.IsEmpty() {
		targetMeta.UpdatedAt = time.Now()
		targetMeta.UpdateName(targetFilename)
		var outputStream io.Reader
		if out != nil {
			outputStream = out.ObjectReader()
		}
		// Update processing status
		updateProcessingState(cObject, manifest)
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
	if err != nil && task.Required {
		return err
	}
	return nil
}

// RegisterConverter interface
func (s *Storage) RegisterConverter(conv converters.Converter) {
	for _, cn := range s.converters {
		if cn == conv {
			return
		}
	}
	s.converters = append(s.converters, conv)
}

// ClearObject metainformation and all subobjects
func (s *Storage) ClearObject(ctx context.Context, obj any) error {
	collectionObject, err := s.Open(ctx, obj)
	if err != nil {
		return err
	}
	ctxlogger.Get(ctx).Info("clean object",
		zap.String("object_bucket", collectionObject.Bucket()),
		zap.String("object_path", collectionObject.Path()))
	return s.driver.Clean(ctx, collectionObject)
}

// ObjectManifest returns global manifest if no present in the object
func (s *Storage) ObjectManifest(ctx context.Context, obj npio.Object) *models.Manifest {
	if obj.Manifest().IsEmpty() {
		if manifestGlobal, _ := s.GetManifest(ctx, obj.Bucket()); manifestGlobal != nil {
			*obj.MustManifest() = *manifestGlobal
			return manifestGlobal
		}
	}
	return obj.Manifest()
}

func (s *Storage) converterByAction(action *models.Action) converters.Converter {
	for _, conv := range s.converters {
		if !conv.Test(action) {
			continue
		}
		return conv
	}
	return nil
}

func (s *Storage) canProcess(task *models.ManifestTask) bool {
	for _, act := range task.Actions {
		for _, conv := range s.converters {
			if conv.Test(act) {
				return true
			}
		}
	}
	return len(task.Actions) == 0
}

func (s *Storage) startProcessing(ctx context.Context, cObject npio.Object) bool {
	s.mx.Lock()
	defer s.mx.Unlock()
	key := processingKey(cObject)
	status, _ := s.processingStatus.Get(ctx, key)
	if models.ObjectStatus(status).IsEmpty() {
		return s.processingStatus.TrySet(ctx, key, status) == nil
	}
	return false
}

func (s *Storage) getProcessingStatus(ctx context.Context, cObject npio.Object) models.ObjectStatus {
	s.mx.Lock()
	defer s.mx.Unlock()
	key := processingKey(cObject)
	v, _ := s.processingStatus.Get(ctx, key)
	if v == "" {
		manifest := s.ObjectManifest(ctx, cObject)
		updateProcessingState(cObject, manifest)
		return cObject.Status()
	}
	return models.ObjectStatus(v)
}

func (s *Storage) setProcessingStatus(ctx context.Context, cObject npio.Object, status models.ObjectStatus) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	cObject.StatusUpdate(status)
	key := processingKey(cObject)
	err := s.processingStatus.Set(ctx, key, status.String())
	return err
}

func processingKey(obj npio.Object) string {
	return fmt.Sprintf("processing_status:%s:%s", obj.Bucket(), obj.Path())
}

func getFinishCloser(conv converters.Converter, in converters.Input, out converters.Output) io.Closer {
	if finisher, _ := conv.(converters.Finisher); finisher != nil {
		return closer(func() error { return finisher.Finish(in, out) })
	} else if out != nil {
		if cl, _ := out.ObjectReader().(io.Closer); cl != nil {
			return cl
		}
	}
	return nil
}
