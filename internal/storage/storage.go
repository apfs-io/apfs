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
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	"github.com/apfs-io/apfs/internal/object"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	"github.com/apfs-io/apfs/internal/storage/processor"
	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/internal/validation"
	"github.com/apfs-io/apfs/models"
)

// Error list...
var (
	ErrStorageInvalidParameterType = errors.New("[storage] invalid parameter type")
	ErrStorageNoConverters         = errors.New("[storage] no any converter registered")
	ErrStorageCollectionIsRequired = errors.New("[storage] collection accessor is required")
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
	driver storio.StorageAccessor

	// Key-value accessor for processing statuses
	processingStatus kvaccessor.KVAccessor

	// Validator runs synchronous checks during Upload.
	// When nil, validation is skipped.
	Validator validation.Validator
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
		processingStatus: opts.processingStatus,
	}
}

// SetWorkflow stores the bucket-level workflow manifest.
func (s *Storage) SetWorkflow(ctx context.Context, group string, w *models.Workflow) error {
	if w == nil {
		return nil
	}
	return s.driver.UpdateWorkflow(ctx, group, w)
}

// GetWorkflow reads the bucket-level workflow manifest.
func (s *Storage) GetWorkflow(ctx context.Context, group string) (*models.Workflow, error) {
	return s.driver.ReadWorkflow(ctx, group)
}

// GetProcessingState returns the current ProcessingState for an object.
func (s *Storage) GetProcessingState(ctx context.Context, objectID string) (*models.ProcessingState, error) {
	return s.driver.ReadState(ctx, storio.ObjectIDType(objectID))
}

// SetProcessingState persists a ProcessingState for an object.
func (s *Storage) SetProcessingState(ctx context.Context, objectID string, state *models.ProcessingState) error {
	return s.driver.WriteState(ctx, storio.ObjectIDType(objectID), state)
}

// ReadMeta reads the Meta for an object (used by the workflow executor).
func (s *Storage) ReadMeta(ctx context.Context, id storio.ObjectID) (*models.Meta, error) {
	obj, err := s.driver.Open(ctx, id)
	if err != nil {
		return nil, err
	}
	return obj.Meta(), nil
}

// WriteMeta persists updated Meta for an object (used by the workflow executor).
func (s *Storage) WriteMeta(ctx context.Context, id storio.ObjectID, meta *models.Meta) error {
	obj, err := s.driver.Open(ctx, id)
	if err != nil {
		return err
	}
	if obj.Meta() != nil {
		*obj.MetaOrNew() = *meta
	}
	return s.UpdateObjectInfo(ctx, obj)
}

// UploadFile into storage
func (s *Storage) UploadFile(ctx context.Context, group string, sourceFilePath string, options ...UploadOption) (storio.Object, error) {
	file, err := os.Open(sourceFilePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return s.Upload(ctx, group, file, options...)
}

// Upload data into storage
func (s *Storage) Upload(ctx context.Context, group string, data io.Reader, options ...UploadOption) (obj storio.Object, err error) {
	if group == "" {
		return nil, ErrStorageInvalidGroupName
	}

	var option uploadOption
	for _, opt := range options {
		opt(&option)
	}

	// Run synchronous pre-upload validation when a Validator is configured.
	if s.Validator != nil {
		req := &validation.ValidationRequest{
			Reader:      data,
			Size:        option.contentLength,
			ContentType: option.contentType,
		}
		if err := s.Validator.Validate(ctx, req); err != nil {
			return nil, err
		}
		// req.Reader may have been wrapped to restore sniffed bytes
		data = req.Reader
	}

	// Create new object container
	obj, err = s.driver.Create(ctx, group, option.customID, option.overwrite, option.Params())
	if err != nil {
		return nil, err
	}

	// Upload object data
	err = s.driver.Update(ctx, obj, models.OriginalFilename, data, nil)
	if err != nil {
		return nil, err
	}

	// Refresh object cached information
	if err = s.UpdateObjectInfo(ctx, obj); err != nil {
		if err2 := s.Delete(ctx, obj); err2 != nil {
			err = fmt.Errorf("%s [%s]", err.Error(), err2.Error())
		}
	}

	if err != nil {
		obj = nil
	}
	return obj, err
}

// Object returns object from storage
func (s *Storage) Object(ctx context.Context, obj any) (storio.Object, error) {
	var nObject storio.Object
	switch val := obj.(type) {
	case string:
		if val == "" {
			return nil, errors.Wrap(ErrStorageInvalidParameterType, "empty object ID")
		}

		// Get object from cache
		objectRecord, err := s.db.Get(val)

		// Check if object is in cache
		if err != nil || objectRecord == nil {
			ctxlogger.Get(ctx).Debug("load object cache", zap.Error(err),
				zap.String("object_id", val))

			// Try to load object from storage
			if nObject, err = s.driver.Open(ctx, storio.ObjectIDType(val)); err != nil {
				return nil, err
			}

			// Refresh object cache
			if objectMode, err := object.ToModel(nObject); err == nil {
				if err = s.db.Set(objectMode); err != nil {
					ctxlogger.Get(ctx).Error("update object object cache", zap.Error(err),
						zap.String("object_id", val))
				}
			}
		} else {
			// If object is in cache then create object from cache
			nObject = object.FromModel(objectRecord)
		}
	case storio.Object:
		// If object is passed as object then just use it
		nObject = val
	default:
		return nil, errors.Wrapf(ErrStorageInvalidParameterType, `type:%T`, obj)
	}

	// Update object status from processing status
	nObject.StatusUpdate(s.getProcessingStatus(ctx, nObject))

	return nObject, nil
}

// OpenObject to read data
func (s *Storage) OpenObject(ctx context.Context, obj any, name string) (storio.Object, io.ReadCloser, error) {
	nObject, err := s.Object(ctx, obj)
	if err != nil {
		return nil, nil, err
	}
	// Read object data
	fr, err := s.driver.Read(ctx, nObject, name)
	if err != nil {
		return nil, nil, err
	}
	return nObject, fr, nil
}

// Delete object completeley
func (s *Storage) Delete(ctx context.Context, obj any, names ...string) (err error) {
	var nObject storio.Object
	if nObject, err = s.Object(ctx, obj); err != nil {
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
func (s *Storage) UpdateObjectInfo(ctx context.Context, obj storio.Object) error {
	mObj, err := object.ToModel(obj)
	if err != nil {
		return err
	}
	return s.db.Set(mObj)
}

// ClearObject metainformation and all subobjects
func (s *Storage) ClearObject(ctx context.Context, obj any) error {
	collectionObject, err := s.Object(ctx, obj)
	if err != nil {
		return err
	}
	ctxlogger.Get(ctx).Info("clean object",
		zap.String("object_bucket", collectionObject.Bucket()),
		zap.String("object_path", collectionObject.Path()))
	return s.driver.Clean(ctx, collectionObject)
}

// ObjectWorkflow returns the workflow for an object.
// If the object has no workflow loaded, it fetches the bucket-level workflow.
func (s *Storage) ObjectWorkflow(ctx context.Context, obj storio.Object) *models.Workflow {
	if obj.Workflow().IsEmpty() {
		if wf, _ := s.GetWorkflow(ctx, obj.Bucket()); wf != nil && !wf.IsEmpty() {
			*obj.WorkflowOrNew() = *wf
			return wf
		}
	}
	return obj.Workflow()
}

// MarkProcessingComplete sets the object status to OK in both the processing-status
// KV (read by every Head call) and the database. Call this once the event pipeline
// determines that all tasks have finished (isComplete=true).
func (s *Storage) MarkProcessingComplete(ctx context.Context, obj storio.Object) error {
	object.TouchUpdatedAt(obj, time.Now())
	s.mx.Lock()
	if err := processor.SetProcessingStatus(ctx, s.processingStatus, obj, models.StatusOK); err != nil {
		s.mx.Unlock()
		return err
	}
	s.mx.Unlock()
	return s.UpdateObjectInfo(ctx, obj)
}

func (s *Storage) getProcessingStatus(ctx context.Context, cObject storio.Object) models.ObjectStatus {
	s.mx.Lock()
	defer s.mx.Unlock()
	return processor.GetProcessingStatus(ctx, s.processingStatus, s, cObject)
}
