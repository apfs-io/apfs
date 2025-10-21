package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	npio "github.com/apfs-io/apfs/internal/io"
	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/internal/storage"
	"github.com/apfs-io/apfs/internal/storage/database"
	"github.com/apfs-io/apfs/internal/storage/processor"
	"github.com/apfs-io/apfs/libs/storerrors"
	"github.com/apfs-io/apfs/models"
)

type bufferItem struct {
	buff []byte
}

// ServiceServer with functional extension
type ServiceServer interface {
	protocol.ServiceAPIServer

	UploadObject(ctx context.Context, group, customID string,
		overwrite bool, tags []string, obj io.Reader) (npio.Object, error)
}

type server struct {
	protocol.UnimplementedServiceAPIServer

	// Amount of stages processed per one iteration
	stageProcessingLimit int

	// Task Processing Limit per one iteration
	taskProcessingLimit int

	// Pull of byte buffers
	bufferpool *sync.Pool

	// Storage object
	store *storage.Storage

	// Processor object
	processor *processor.Processor

	// Event stream object chanel
	eventStream nc.Publisher

	// Update state accessor
	updateState updateStateI
}

// NewServer object which implements RPC actions
func NewServer(ctx context.Context, connect, storageConnect, stateConnect string, opts ...Option) (ServiceServer, error) {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}
	database, err := database.Open(ctx, connect)
	if err != nil {
		return nil, err
	}
	driver, err := newStorage(ctx, storageConnect)
	if err != nil {
		return nil, err
	}
	stateKV, err := newKVAccessor(stateConnect)
	if err != nil {
		return nil, err
	}
	pool := &sync.Pool{New: func() any {
		return &bufferItem{buff: make([]byte, 10*1024)}
	}}
	return &server{
		stageProcessingLimit: options.stageProcessingLimit,
		taskProcessingLimit:  options.taskProcessingLimit,
		bufferpool:           pool,
		eventStream:          options.eventStream,
		updateState:          options.updateState,
		store:                options._storage(database, driver, stateKV),
		processor:            options._processor(driver, stateKV),
	}, nil
}

// Head returns basic meta information from object
func (s *server) Head(ctx context.Context, obj *protocol.ObjectID) (*protocol.SimpleObjectResponse, error) {
	ctxlogger.Get(ctx).Info("Object HEAD", zap.String("object_id", obj.GetId()))

	// Get object descriptor
	sObject, err := s.store.Object(ctx, obj.GetId())
	if err != nil && !storerrors.IsNotFound(err) {
		return &protocol.SimpleObjectResponse{
			Status:  protocol.ResponseStatusCode_FAILED,
			Message: err.Error(),
		}, nil
	}
	if sObject == nil || storerrors.IsNotFound(err) {
		return &protocol.SimpleObjectResponse{
			Status:  protocol.ResponseStatusCode_NOT_FOUND,
			Message: "Not found",
		}, nil
	}

	// Get transformation manifest of the object
	objectManifest := s.store.ObjectManifest(ctx, sObject)

	// If object is not completed or have extra objects then initiate new update task
	if !sObject.Meta().IsConsistent(objectManifest) && s.updateState.TryBeginUpdate(obj.GetId()) {
		s.updateObjectState(ctx, sObject.ID().String())
	}

	object, err := s.protoObject(sObject)
	if err != nil {
		return &protocol.SimpleObjectResponse{
			Status:  protocol.ResponseStatusCode_FAILED,
			Message: err.Error(),
		}, err
	}

	return &protocol.SimpleObjectResponse{
		Status:  protocol.ResponseStatusCode_OK,
		Message: "Object successfully loaded",
		Object:  object,
	}, nil
}

// Refresh pbject processing (recreate thumbs, meta data and etc.)
func (s *server) Refresh(ctx context.Context, obj *protocol.ObjectID) (*protocol.SimpleResponse, error) {
	ctxlogger.Get(ctx).Info("Refresh PUT", zap.String("object_id", obj.GetId()))

	sObject, err := s.store.Object(ctx, obj.GetId())
	if err != nil && !storerrors.IsNotFound(err) {
		return &protocol.SimpleResponse{
			Status:  protocol.ResponseStatusCode_FAILED,
			Message: err.Error(),
		}, nil
	}
	if sObject == nil || storerrors.IsNotFound(err) {
		return &protocol.SimpleResponse{
			Status:  protocol.ResponseStatusCode_NOT_FOUND,
			Message: "Not found",
		}, nil
	}

	// Get transformation manifest of the object
	s.refreshObjectState(ctx, sObject.ID().String())

	return &protocol.SimpleResponse{
		Status:  protocol.ResponseStatusCode_OK,
		Message: "Object successfully refreshed",
	}, nil
}

// Get object with content by stream
func (s *server) Get(obj *protocol.ObjectID, stream protocol.ServiceAPI_GetServer) error {
	ctx := stream.Context()
	ctxlogger.Get(ctx).Info("Object GET", zap.String("object_id", obj.GetId()))

	sObject, err := s.store.Object(ctx, obj.GetId())
	if err != nil {
		return err
	}

	// Get transformation manifest of the object
	objectManifest := s.store.ObjectManifest(ctx, sObject)

	// If object is not completed or have extra objects then initiate new update task
	if !sObject.Meta().IsConsistent(objectManifest) && s.updateState.TryBeginUpdate(obj.GetId()) {
		s.updateObjectState(ctx, sObject.ID().String())
	}

	var (
		data    io.ReadCloser
		openErr error
		fileTry = map[string]bool{}
	)

	// Try to open one of the object names
	for _, name := range obj.Name {
		if fileTry[name] {
			continue
		}
		fileTry[name] = true
		if _, data, err = s.store.OpenObject(ctx, sObject, name); err == nil {
			break
		}
		if !storerrors.IsNotFound(err) {
			return err
		}
		openErr = multierr.Append(openErr, err)
	}
	if err != nil {
		return openErr
	}
	defer func() { _ = data.Close() }()

	var (
		buf    = s.allocBuffer()
		res    *protocol.ObjectResponse
		object *protocol.Object
	)
	defer s.releaseBuffer(buf)

	if object, err = s.protoObject(sObject); err != nil {
		return err
	}

	// Send initial object info...
	res = &protocol.ObjectResponse{
		Object: &protocol.ObjectResponse_Response{
			Response: &protocol.SimpleObjectResponse{
				Status:  protocol.ResponseStatusCode_OK,
				Message: "Object successfully loaded",
				Object:  object,
			},
		},
	}

	// Send headers...
	if err = stream.Send(res); err != nil {
		return err
	}

	// Content data item
	responseItem := &protocol.ObjectResponse{
		Object: &protocol.ObjectResponse_Content{
			Content: &protocol.DataContent{},
		},
	}

	for {
		// put as many bytes as `chunkSize` into the buf array.
		n, err := data.Read(buf.buff)
		if n > 0 {
			responseItem.Object.(*protocol.ObjectResponse_Content).
				Content.Content = buf.buff[:n]
			err = stream.Send(responseItem)
		}
		if err != nil && errors.Is(err, io.EOF) {
			return err
		}
		if n < 1 || errors.Is(err, io.EOF) {
			break
		}
	}

	return nil
}

// SetManifest of the group which will be applied for entities of the `group`
// - {storage-domain}/{group}/manifest.json
//   - {storage-domain}/{group}/{object-codename}/manifest.json - reference to `{storage-domain}/{group}/manifest.json`
func (s *server) SetManifest(ctx context.Context, manifest *protocol.DataManifest) (_ *protocol.SimpleResponse, err error) {
	ctxlogger.Get(ctx).Info("Set Manifest",
		zap.String("manifest_group", manifest.GetGroup()),
		zap.Any("manifest", manifest.Manifest),
	)

	// Convert manifest from proto to model
	manifestObj := manifest.Manifest.ToModel()
	manifestObj.PrepareInfo()

	// Save manifest into storage and update all objects (lazy by request) with the same group
	err = s.store.SetManifest(ctx, manifest.GetGroup(), manifestObj)
	if err != nil {
		ctxlogger.Get(ctx).Error("Set Manifest",
			zap.String("manifest_group", manifest.GetGroup()),
			zap.Error(err))
		return &protocol.SimpleResponse{
			Status:  protocol.ResponseStatusCode_FAILED,
			Message: fmt.Sprintf("Manifest [%s] setup error: %s", manifest.GetGroup(), err.Error()),
		}, err
	}

	return &protocol.SimpleResponse{
		Status:  protocol.ResponseStatusCode_OK,
		Message: fmt.Sprintf("Manifest [%s] was setuped", manifest.GetGroup()),
	}, nil
}

// GetManifest from the group
func (s *server) GetManifest(ctx context.Context, group *protocol.ManifestGroup) (_ *protocol.ManifestResponse, err error) {
	ctxlogger.Get(ctx).Info("Get Manifest",
		zap.String("manifest_group", group.GetGroup()))

	defer func() {
		if err != nil {
			ctxlogger.Get(ctx).Error("Get Manifest",
				zap.String("manifest_group", group.GetGroup()),
				zap.Error(err))
		}
	}()

	manifest, err := s.store.GetManifest(ctx, group.GetGroup())
	if err != nil {
		status := protocol.ResponseStatusCode_FAILED
		if storerrors.IsNotFound(err) {
			status = protocol.ResponseStatusCode_NOT_FOUND
		}
		return &protocol.ManifestResponse{
			Status:  status,
			Message: fmt.Sprintf("Manifest [%s] get error: %s", group.GetGroup(), err.Error()),
		}, err
	}

	manifestProto, err := protocol.ManifestFromModel(manifest)
	if err != nil {
		return &protocol.ManifestResponse{
			Status:  protocol.ResponseStatusCode_FAILED,
			Message: fmt.Sprintf("Manifest [%s] converting error: %s", group.GetGroup(), err.Error()),
		}, err
	}

	return &protocol.ManifestResponse{
		Status:   protocol.ResponseStatusCode_OK,
		Manifest: manifestProto,
	}, nil
}

// Upload new object from the stream
func (s *server) Upload(stream protocol.ServiceAPI_UploadServer) (err error) {
	var (
		tmpfile   *os.File
		file      npio.Object
		group     string
		customID  string
		overwrite bool
		tags      []string
		object    *protocol.Object
		ctx       = stream.Context()
	)

	ctxlogger.Get(ctx).Info("Upload")

	defer func() {
		if nerr := recover(); nerr != nil {
			ctxlogger.Get(ctx).Error("Upload::recover",
				zap.String("group", group),
				zap.String("object_id", object.GetId()),
				zap.String("custom_object_id", customID),
				zap.Any("recover_err", nerr),
				zap.Error(err))
		} else if err != nil {
			ctxlogger.Get(ctx).Error("Upload",
				zap.String("group", group),
				zap.String("object_id", object.GetId()),
				zap.String("custom_object_id", customID),
				zap.Error(err))
		}
	}()

	if tmpfile, err = os.CreateTemp("", "upload"); err != nil {
		return err
	}

	defer func() {
		_ = tmpfile.Close()
		_ = os.Remove(tmpfile.Name()) // clean up
	}()

	// while there are messages coming
	for {
		var data *protocol.Data
		if data, err = stream.Recv(); err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrapf(err, "failed unexpectadely while reading chunks from stream")
		}
		if info := data.GetInfo(); info != nil {
			group = info.GetGroup()
			customID = info.GetCustomId()
			overwrite = info.GetOverwrite()
			ctxlogger.Get(ctx).Info("Upload",
				zap.String("custom_object_id", customID),
				zap.Bool("overwrite", overwrite),
				zap.String("group", group))
		}
		if t := data.GetTags(); len(t) > 0 {
			tags = t
		}
		if cont := data.GetContent(); cont != nil {
			if _, err = tmpfile.Write(cont.Content); err != nil {
				return err
			}
		}
	}

	// Reset file cursor
	if _, err = tmpfile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// Upload file into store
	file, err = s.store.Upload(
		stream.Context(),
		group,
		tmpfile,
		storage.WithTags(tags),
		storage.WithCustomID(
			npio.ObjectIDType(customID),
		),
		storage.WithOverwrite(overwrite),
	)
	if err == nil {
		object, err = s.protoObject(file)

		// once the transmission finished, send the
		// confirmation if nothign went wrong
		if err == nil {
			err = stream.SendAndClose(&protocol.SimpleObjectResponse{
				Status:  protocol.ResponseStatusCode_OK,
				Message: "Upload received with success",
				Object:  object,
			})
		}
	}

	if err != nil {
		_ = stream.SendAndClose(&protocol.SimpleObjectResponse{
			Status:  protocol.ResponseStatusCode_FAILED,
			Message: "Upload failed: " + err.Error(),
			Object:  nil,
		})
	}

	s.sendEvent(ctx, models.UpdateEventType, object.ToModel(), err)
	return err
}

// UploadObject for the specific group of files
func (s *server) UploadObject(ctx context.Context, group, customID string, overwrite bool, tags []string, obj io.Reader) (npio.Object, error) {
	var (
		err    error
		nobj   npio.Object
		object *protocol.Object
	)
	ctxlogger.Get(ctx).Error("UploadObject", zap.String("group", group))

	defer func() {
		if err != nil {
			ctxlogger.Get(ctx).Error("UploadObject",
				zap.String("group", group),
				zap.String("object_id", object.GetId()),
				zap.Error(err))
		}
	}()
	// Upload file into store
	nobj, err = s.store.Upload(ctx, group, obj,
		storage.WithTags(tags),
		storage.WithCustomID(
			npio.ObjectIDType(customID),
		),
		storage.WithOverwrite(overwrite),
	)
	if err == nil {
		object, err = s.protoObject(nobj)
	}
	s.sendEvent(ctx, models.UpdateEventType, object.ToModel(), err)
	return nobj, err
}

// Delete object or subobjects
func (s *server) Delete(ctx context.Context, obj *protocol.ObjectIDNames) (_ *protocol.SimpleResponse, err error) {
	ctxlogger.Get(ctx).Info("DELETE", zap.String("object_id", obj.GetId()))
	defer func() {
		if err != nil {
			ctxlogger.Get(ctx).Error("delete",
				zap.String("object_id", obj.GetId()),
				zap.Strings("names", obj.Names),
				zap.Error(err))
		}
	}()
	if err = s.store.Delete(ctx, obj.GetId(), obj.Names...); err != nil {
		return nil, err
	}
	s.sendEvent(ctx, models.DeleteEventType, &models.Object{ID: obj.GetId()}, err)
	return &protocol.SimpleResponse{
		Status:  protocol.ResponseStatusCode_OK,
		Message: fmt.Sprintf("Object %s was deleted", obj.GetId()),
	}, nil
}

// Receive method of stream processing pipeline
func (s *server) Receive(message nc.Message) error {
	var (
		err     error
		ctx     = message.Context()
		event   models.Event
		cObject npio.Object
	)

	// Unpack event from body
	if err := json.Unmarshal(message.Body(), &event); err != nil {
		ctxlogger.Get(ctx).Error("event unmarshal", zap.Error(err))
		return message.Ack()
	}

	fields := []zapcore.Field{zap.String("event", event.Type.String())}

	if event.Object != nil {
		fields = append(fields,
			zap.Reflect("object_id", event.Object.ObjectID()),
			zap.Reflect("object_bucket", event.Object.Bucket),
			zap.Reflect("object_path", event.Object.Path),
		)
	} else {
		fields = append(fields,
			zap.Reflect("object_id", "invalid"),
			zap.String("event_body", string(message.Body())),
		)
	}

	ctxlogger.Get(ctx).Debug("run processed object", fields...)

	// Preload object for event type
	if event.Type == models.RefreshEventType || event.Type == models.UpdateEventType {
		cObject, err = s.store.Object(ctx, event.Object.ObjectID())
		if err != nil {
			isNotFound := storerrors.IsNotFound(err)
			ctxlogger.Get(ctx).Error("process",
				append(fields, zap.Error(err), zap.Bool(`not_found`, isNotFound))...)
			if !isNotFound {
				// send processed error event
				if event.Object != nil {
					_ = event.Object.Manifest.SetValue(cObject.Manifest())
					_ = event.Object.Meta.SetValue(cObject.Meta())
				}
				s.sendEvent(ctx, models.ProcessedEventType, event.Object, err)
			} else {
				s.sendEvent(ctx, models.DeleteEventType, event.Object, err)
			}
			return message.Ack()
		}
	}

	switch event.Type {
	case models.RefreshEventType:
		cObject.Meta().ResetCompletion()
		fallthrough
	case models.UpdateEventType:
		s.updateEventAction(ctx, &event, cObject, fields)
	case models.ProcessedEventType:
		ctxlogger.Get(ctx).Info("processed object", fields...)
	case models.DeleteEventType:
		ctxlogger.Get(ctx).Info("delete object", fields...)
	default:
		ctxlogger.Get(ctx).Error("undefined event type", fields...)
	}
	return message.Ack()
}

func (s *server) updateEventAction(ctx context.Context, event *models.Event, cObject npio.Object, fields []zapcore.Field) {
	var (
		err        error
		isComplete bool
		items      []*models.ItemMeta
	)
	cObject.Meta().RemoveExcessTasks(cObject.Manifest())

	// Clean object before continue
	if cObject.Meta().Main.IsEmpty() {
		if err = s.store.ClearObject(ctx, cObject); err != nil {
			ctxlogger.Get(ctx).Error("clear object", append(fields, zap.Error(err),
				zap.Bool(`not_found`, storerrors.IsNotFound(err)))...)
		}
	}

	if cObject.Meta().Main.IsEmpty() {
		ctxlogger.Get(ctx).Error("invalid processing object", append(fields, zap.Error(err))...)
		return
	}
	if event.Type == models.RefreshEventType {
		items = cObject.Meta().Items
	} else {
		items = cObject.Meta().ExcessItems(cObject.Manifest())
	}
	// Remove redundant extra objects
	_ = s.removeObjectItems(ctx, cObject, items, fields...)
	// Process next task actions
	isComplete, err = s.processor.ProcessTasks(ctx, cObject,
		s.taskProcessingLimit, s.taskProcessingLimit)
	switch {
	case err != nil:
		isNotFound := storerrors.IsNotFound(err)
		ctxlogger.Get(ctx).Error("process",
			append(fields, zap.Error(err), zap.Bool(`not_found`, isNotFound))...)
		// send processed error event
		if !isNotFound {
			s.sendEvent(ctx, models.UpdateEventType, event.Object, err)
		} else {
			s.sendEvent(ctx, models.DeleteEventType, event.Object, err)
		}
	case isComplete:
		ctxlogger.Get(ctx).Info("process complete", fields...)
		if event.Object != nil {
			_ = event.Object.Manifest.SetValue(cObject.Manifest())
			_ = event.Object.Meta.SetValue(cObject.Meta())
		}
		s.sendEvent(ctx, models.ProcessedEventType, event.Object, nil)
	default:
		ctxlogger.Get(ctx).Info("next step", fields...)
		s.sendEvent(ctx, models.UpdateEventType, event.Object, nil)
	}
}

func (s *server) removeObjectItems(ctx context.Context, cObject npio.Object,
	items []*models.ItemMeta, fields ...zapcore.Field) error {
	if len(items) == 0 {
		return nil
	}
	removeSubObjects := make([]string, 0, len(items))
	for _, it := range items {
		removeSubObjects = append(removeSubObjects, it.Fullname())
	}
	err := s.store.Delete(ctx, cObject, removeSubObjects...)
	if err != nil {
		ctxlogger.Get(ctx).Error(`delete old objects`,
			zap.Error(err), zap.Strings(`objects`, removeSubObjects))
	} else {
		ctxlogger.Get(ctx).Debug("refresh delete old",
			append(fields, zap.Strings(`objects`, removeSubObjects))...)
	}
	cObject.Meta().RemoveExcessTasks(cObject.Manifest())
	return err
}

func (s *server) protoObject(obj npio.Object) (_ *protocol.Object, err error) {
	var manifest *protocol.Manifest
	if obj.Manifest() != nil {
		if manifest, err = protocol.ManifestFromModel(obj.Manifest()); err != nil {
			return nil, err
		}
	}
	return &protocol.Object{
		Id:     obj.ID().String(),
		Bucket: obj.Bucket(),
		Path:   obj.Path(),
		HashId: obj.MustMeta().Main.HashID,

		Status: &protocol.ObjectStatus{
			Status:  obj.Status().String(),
			Message: obj.StatusMessage(),
		},

		ObjectType:  string(obj.MustMeta().Main.Type),
		ContentType: obj.MustMeta().Main.ContentType,
		Manifest:    manifest,
		Meta:        protocol.MetaFromModel(obj.MustMeta()),
		Size:        uint32(obj.MustMeta().Main.Size),

		CreatedAt: obj.CreatedAt().UnixNano(),
		UpdatedAt: obj.UpdatedAt().UnixNano(),
	}, nil
}

func (s *server) allocBuffer() *bufferItem {
	return s.bufferpool.Get().(*bufferItem)
}

func (s *server) releaseBuffer(buff *bufferItem) {
	if buff == nil {
		return
	}
	// buff.buff = buff.buff[:0]
	s.bufferpool.Put(buff)
}

func (s *server) errorLog(ctx context.Context, err error, args ...any) {
	if err != nil {
		ctxlogger.Get(ctx).Sugar().Error(append(args, err)...)
	}
}

func (s *server) refreshObjectState(ctx context.Context, objectID string) {
	ctxlogger.Get(ctx).Info(`Refresh object state`, zap.String(`object_id`, objectID),
		zap.String(`action`, models.RefreshEventType.String()))
	s.sendEvent(ctx, models.RefreshEventType, &models.Object{ID: objectID}, nil)
}

func (s *server) updateObjectState(ctx context.Context, objectID string) {
	ctxlogger.Get(ctx).Info(`Update object state`, zap.String(`object_id`, objectID),
		zap.String(`action`, models.UpdateEventType.String()))
	s.sendEvent(ctx, models.UpdateEventType, &models.Object{ID: objectID}, nil)
}

func (s *server) sendEvent(ctx context.Context, etype models.EventType, obj *models.Object, err error) {
	var emsg string
	if err != nil {
		emsg = err.Error()
	}
	ctxlogger.Get(ctx).Info("sendEvent",
		zap.String("event_type", etype.String()),
		zap.String("object_id", obj.ObjectID()))
	s.errorLog(ctx, s.eventStream.Publish(ctx, &models.Event{
		Type:   etype,
		Error:  emsg,
		Object: obj,
	}))
}
