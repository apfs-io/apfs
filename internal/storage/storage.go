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

	"go.uber.org/zap"

	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/object"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	"github.com/apfs-io/apfs/internal/storage/processor"
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

	// Key-value accessor for processing statuses
	processingStatus kvaccessor.KVAccessor
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
func (s *Storage) Object(ctx context.Context, obj any) (npio.Object, error) {
	var nObject npio.Object
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
				zap.String("object_id", val),
				zap.Stack("stack"))

			// Try to load object from storage
			if nObject, err = s.driver.Open(ctx, npio.ObjectIDType(val)); err != nil {
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
	case npio.Object:
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
func (s *Storage) OpenObject(ctx context.Context, obj any, name string) (npio.Object, io.ReadCloser, error) {
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
	var nObject npio.Object
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
func (s *Storage) UpdateObjectInfo(ctx context.Context, obj npio.Object) error {
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

func (s *Storage) getProcessingStatus(ctx context.Context, cObject npio.Object) models.ObjectStatus {
	s.mx.Lock()
	defer s.mx.Unlock()
	return processor.GetProcessingStatus(ctx, s.processingStatus, s, cObject)
}
