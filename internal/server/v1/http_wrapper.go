package v1

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/demdxx/gocast/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/libs/storerrors"
)

// ServerHTTPWrapper object
type ServerHTTPWrapper struct {
	*server
}

// NewHTTPWrapper returns HTTP wrapper
func NewHTTPWrapper(s any) *ServerHTTPWrapper {
	return &ServerHTTPWrapper{server: s.(*server)}
}

// UploadHTTPHandler defined HTTP upload handler
// query params:
//
//	id:string      - custom object ID
//	overwrite:bool - overwrite with custom object ID
func (s *ServerHTTPWrapper) UploadHTTPHandler(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		customID  = r.URL.Query().Get("id")
		overwrite = gocast.Bool(r.URL.Query().Get("overwrite"))
		group     = chi.URLParam(r, "group")
	)
	if group == "" {
		group = r.URL.Query().Get("group")
	}

	tags := r.URL.Query()["tags"]
	if err := r.ParseForm(); err != nil {
		ctxlogger.Get(ctx).Error("parse request form", zap.Error(err))
		errorResponse(w, "parse request error: "+err.Error())
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		ctxlogger.Get(ctx).Error("parse request body", zap.Error(err))
		errorResponse(w, "parse request body error: "+err.Error())
		return
	}

	filePart, err := reader.NextPart()
	if err != nil {
		ctxlogger.Get(ctx).Error("parse request body part", zap.Error(err))
		errorResponse(w, "parse request body part error: "+err.Error())
		return
	}

	// Close all parts after finish
	defer func() {
		_ = filePart.Close()

		// Close all parts
		for {
			filePart, err = reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				ctxlogger.Get(ctx).Error("next parse", zap.Error(err))
				break
			} else {
				_ = filePart.Close()
			}
		}
	}()

	nobj, err := s.server.UploadObject(ctx, group, customID, overwrite, tags, filePart)
	if err != nil {
		ctxlogger.Get(ctx).Error("upload to storage", zap.Error(err))
		errorResponse(w, "upload to storage error: "+err.Error())
		return
	}

	pobj, err := s.protoObject(nobj)
	if err != nil {
		ctxlogger.Get(ctx).Error("upload to storage", zap.Error(err))
		errorResponse(w, "object convert error: "+err.Error())
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(&protocol.SimpleObjectResponse{
		Status: protocol.ResponseStatusCode_RESPONSE_STATUS_CODE_OK,
		Object: pobj,
	})
}

// GetHTTPHandler defined HTTP object read handler
func (s *ServerHTTPWrapper) GetHTTPHandler(w http.ResponseWriter, r *http.Request) {
	s._getHTTPHandler(w, r, false)
}

// HeadHTTPHandler defined HTTP object info handler
func (s *ServerHTTPWrapper) HeadHTTPHandler(w http.ResponseWriter, r *http.Request) {
	s._getHTTPHandler(w, r, true)
}

func (s *ServerHTTPWrapper) _getHTTPHandler(w http.ResponseWriter, r *http.Request, headOnly bool) {
	var (
		ctx   = r.Context()
		id    = chi.URLParam(r, "*")
		query = r.URL.Query()
		name  = query.Get("name")
	)
	if id == "" {
		id = query.Get("id")
	}
	ctxlogger.Get(ctx).Info("Object GET", zap.String("object_id", id))

	// Get object reference by ID
	sObject, err := s.store.Object(ctx, id)
	if err != nil && !storerrors.IsNotFound(err) {
		ctxlogger.Get(ctx).Error("open object link by ID",
			zap.String("object_id", id),
			zap.String("object_name", name),
			zap.Error(err))
		errorResponse(w, err.Error())
		return
	}
	if sObject == nil || storerrors.IsNotFound(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Get transformation manifest of the object
	objectManifest := s.store.ObjectManifest(ctx, sObject)

	// If object is not completed or have extra objects then initiate new update task
	if !sObject.Meta().IsConsistent(objectManifest) && s.updateState.TryBeginUpdate(id) {
		s.updateObjectState(ctx, sObject.ID().String())
	}

	var data io.ReadCloser
	if !headOnly {
		if _, data, err = s.store.OpenObject(ctx, sObject, name); err != nil {
			ctxlogger.Get(ctx).Error("open object by ID",
				zap.String("object_id", id),
				zap.String("object_name", name),
				zap.Error(err))
			errorResponse(w, err.Error())
			return
		}
		defer func() { _ = data.Close() }()
	}

	if sObject.IsOriginal(name) {
		name = sObject.PrepareName(name)
	}
	contentType := mime.TypeByExtension(filepath.Ext(name))
	sObjectMeta := sObject.MustMeta()
	w.Header().Add("Content-Type", contentType)
	w.Header().Add(
		gocast.IfThen(headOnly, "X-Content-Size", "Content-Length"),
		gocast.Str(sObjectMeta.ItemByName(name).Size))
	if headOnly || gocast.Bool(query.Get("meta")) {
		w.Header().Add("X-Content-Meta", encodeJSONBase64(sObjectMeta))
	}
	if len(sObject.MustMeta().Tags) > 0 {
		w.Header().Add("X-Content-Tags", strings.Join(sObject.MustMeta().Tags, ","))
	}
	w.WriteHeader(http.StatusOK)

	if data != nil {
		if _, err := io.Copy(w, data); err != nil {
			ctxlogger.Get(ctx).Error("write response data",
				zap.String("object_id", id),
				zap.String("object_name", name),
				zap.Error(err))
		}
	}
}

func errorResponse(w http.ResponseWriter, err string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(&protocol.SimpleObjectResponse{
		Status:  protocol.ResponseStatusCode_RESPONSE_STATUS_CODE_FAILED,
		Message: err,
	})
}

func encodeJSONBase64(obj any) string {
	data, _ := json.Marshal(obj)
	return base64.StdEncoding.EncodeToString(data)
}
