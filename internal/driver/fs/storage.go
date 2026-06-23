// Package fs implements file storage collection on file (disk) system
package fs

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	"github.com/apfs-io/apfs/internal/object"
	datalib "github.com/apfs-io/apfs/internal/storage/data"
	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/internal/storio/objectpath"
	"github.com/apfs-io/apfs/internal/utils"
	workflowparser "github.com/apfs-io/apfs/internal/workflow"
	"github.com/apfs-io/apfs/libs/storerrors"
	"github.com/apfs-io/apfs/models"
)

const (
	metaFileName = "meta.json"
)

// Error set...
var (
	ErrObjectIDIsNotValid                 = errors.New("object ID is not valid")
	ErrCollectionFilePathMustBeDirectory  = errors.New("filepath must be directory")
	ErrCollectionFileOriginalFilepath     = errors.New("invalid original filepath")
	ErrCollectionFileNotDefinedInManifest = errors.New("object not defined in the manifest")
	ErrUnsupportedContentType             = errors.New("content-type is not supported")
	ErrInvalidRootDirectory               = errors.New("invalid root directory")
	ErrCustomObjectIDIsNotValid           = errors.New("custom object ID is not valid or taken")
)

// Storage on file-system and remove files
type Storage struct {
	root string

	// Generator of the path on the file system
	pathgen objectpath.Generator
	fcache  FileCacher
	mcache  FileCacher
}

// NewStorage of objects on FS
func NewStorage(root string, options ...Option) (*Storage, error) {
	if isSystemDir(root) {
		return nil, errors.Wrap(ErrInvalidRootDirectory, root)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	store := &Storage{root: root}
	for _, opt := range options {
		opt(store)
	}
	if store.fcache == nil {
		store.fcache = &dummyFileCache{}
	}
	if store.mcache == nil {
		store.mcache = &dummyFileCache{}
	}
	if store.pathgen == nil {
		store.pathgen = objectpath.NewBasePathgenerator(
			"{{year}}/{{month}}/{{md5:1}}/{{md5:2}}/{{md5}}",
			objectpath.WithChecker(PathChecker),
		)
	}
	return store, nil
}

// ReadWorkflow reads the bucket-level workflow manifest from manifest.yaml.
func (c *Storage) ReadWorkflow(ctx context.Context, bucket string) (*models.Workflow, error) {
	yamlPath := filepath.Join(c.root, bucket, "manifest.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.Workflow{}, nil
		}
		return nil, err
	}
	return workflowparser.ParseWorkflow(data)
}

// UpdateWorkflow writes the bucket-level workflow manifest as manifest.yaml.
func (c *Storage) UpdateWorkflow(ctx context.Context, bucket string, workflow *models.Workflow) error {
	if workflow == nil {
		return nil
	}
	data, err := workflow.MarshalYAML()
	if err != nil {
		return err
	}
	dir := filepath.Join(c.root, bucket)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.yaml"), data, 0o644)
}

// Create new file object
func (c *Storage) Create(ctx context.Context, bucket string, id storio.ObjectID, overwrite bool, params url.Values) (storio.Object, error) {
	ctxlogger.Get(ctx).Info("Create", zap.Any("id", id))

	path, err := c.newPath(bucket, id, !overwrite)
	if err != nil {
		return nil, err
	}

	// Init new object container
	obj := object.NewObject(
		storio.ObjectIDType(filepath.Join(bucket, path)), bucket, path)

	if overwrite {
		if err := c.Remove(ctx, obj); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	// Load workflow from bucket-level manifest.yaml
	if err := c.loadWorkflow(ctx, obj); err != nil {
		return nil, err
	}

	// Update meta information of the object
	if len(params) > 0 {
		tags := params["tags"]
		params.Del("tags")
		meta := obj.MetaOrNew()
		meta.Tags = tags
		meta.Params = params
		if err = c.saveMeta(obj); err != nil {
			if err := c.Remove(ctx, obj); err != nil {
				ctxlogger.Get(ctx).Error(`remove file`,
					zap.String(`object_id`, obj.ID().String()),
					zap.Error(err))
			}
			return nil, err
		}
	}

	// Save object information
	updateObjectFileInfo(obj, nil)

	return obj, err
}

// UpdateParams in the object. If name is present then update only params linked with the subobject
func (c *Storage) UpdateParams(ctx context.Context, id storio.ObjectID, params url.Values) error {
	obj, err := c._ID2Object(ctx, id)
	if err != nil {
		return err
	}
	meta := obj.MetaOrNew()
	if params != nil {
		tags := params["tags"]
		params.Del("tags")
		meta.Tags = tags
		meta.Params = params
	} else {
		meta.Tags = nil
		meta.Params = nil
	}
	return c.saveMeta(obj)
}

// Scan storage by pattern
//
//	pattern: search type equals to glob https://golang.org/pkg/path/filepath/#Glob
func (c *Storage) Scan(ctx context.Context, pattern string, walkf storio.WalkStorageFunc) error {
	return filepath.Walk(c.root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if ok, _ := filepath.Match(pattern, path); !ok {
			return nil
		}
		return walkf(path, nil)
	})
}

// Open exixting file
func (c *Storage) Open(ctx context.Context, id storio.ObjectID) (_ storio.Object, err error) {
	var (
		info     os.FileInfo
		object   = objectFromID(id)
		fullpath = c.fullpath(object)
	)

	if info, err = os.Stat(fullpath); os.IsNotExist(err) {
		return nil, storerrors.WrapNotFound(fullpath, err)
	} else if err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, ErrCollectionFilePathMustBeDirectory
	}

	// Load workflow from bucket-level manifest.yaml
	if err = c.loadWorkflow(ctx, object); err != nil {
		return nil, errors.Wrap(err, "manifest.yaml")
	}

	// Load meta information
	if err = c.loadMeta(object); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Update object information
	updateObjectFileInfo(object, info)

	return object, nil
}

func (c *Storage) _ID2Object(ctx context.Context, id storio.ObjectID) (storio.Object, error) {
	switch t := id.(type) {
	case storio.Object:
		return t, nil
	default:
		return c.Open(ctx, id)
	}
}

// Update data in the storage
func (c *Storage) Update(ctx context.Context, id storio.ObjectID, name string, data io.Reader, meta *models.ItemMeta) error {
	var (
		fullname string
		head     []byte
		ndata    io.Reader
		obj, err = c._ID2Object(ctx, id)
	)
	if err != nil {
		return err
	}

	// Remove old item file
	if err = c.Remove(ctx, obj, name); err != nil {
		return err
	}

	if fullname, err = c.FileFullname(obj, name); err != nil {
		return err
	}

	// Read head information of file before store
	if head, ndata, err = datalib.ReadHead(data); err != nil {
		return err
	}

	var (
		contentType = datalib.ContentTypeByData(head)
		fileExt     = filepath.Ext(name)
	)

	if fileExt == "" {
		fileExt = datalib.ExtensionByContentType(contentType)
	}

	if !obj.Workflow().IsValidContentType(contentType) {
		return errors.Wrap(ErrUnsupportedContentType, contentType)
	}

	fullname, fileExt = prepareFileExt(fullname, fileExt)
	fileFullpath := filepath.Join(c.fullpath(obj), fullname+fileExt)

	// Save data to file
	if err = datalib.SaveDataToFile(ndata, fileFullpath); err != nil {
		return err
	}

	// Collect file information
	if meta, err = utils.CollectFileInfo(meta, fileFullpath, contentType); err != nil {
		return err
	}

	// Collect information
	if meta.NameExt == `` {
		meta.NameExt = strings.TrimPrefix(fileExt, ".")
	}
	return c.saveObjectMeta(ctx, obj, name, meta)
}

// UpdateMeta data of the specific subobject in the storage
func (c *Storage) UpdateMeta(ctx context.Context, id storio.ObjectID, name string, meta *models.ItemMeta) error {
	obj, err := c._ID2Object(ctx, id)
	if err != nil {
		return err
	}
	return c.saveObjectMeta(ctx, obj, name, meta)
}

// Remove file from directory by name without extension of file
func (c *Storage) Remove(ctx context.Context, id storio.ObjectID, names ...string) error {
	if !isValidID(id) {
		return ErrObjectIDIsNotValid
	}
	if len(names) > 0 {
		return c.removeFiles(ctx, id, names...)
	}
	objPath := c.fullpath(id)
	err := os.RemoveAll(objPath)
	if err != nil {
		return err
	}
	for {
		objPath = filepath.Dir(objPath)
		if !isEmptyDir(objPath) || !strings.HasPrefix(objPath, c.root) || len(objPath) <= len(c.root)+1 {
			break
		}
		if os.Remove(objPath) != nil {
			ctxlogger.Get(ctx).Error("remove empty dir", zap.String("dir", objPath))
			break
		}
	}
	return err
}

// Read returns new reader from filepath
func (c *Storage) Read(ctx context.Context, id storio.ObjectID, name string) (io.ReadCloser, error) {
	var (
		obj, err = c._ID2Object(ctx, id)
		fullname string
		nfile    io.ReadCloser
	)
	if err != nil {
		return nil, errors.Wrap(err, `object read`)
	}
	if fullname, err = c.FileFullname(obj, name); err != nil {
		return nil, err
	}
	if nfile, err = c.fcache.Read(filepath.Join(c.fullpath(obj), fullname)); err != nil {
		if os.IsNotExist(err) {
			return nil, storerrors.WrapNotFound(c.fullpath(obj), err)
		}
		return nil, err
	}
	return nfile, err
}

// FileFullname resolves the full filename (with extension) for a named subfile.
// For non-original names it first checks meta.Items, then falls back to the
// bare name. Returns ErrCollectionFileNotDefinedInManifest if neither meta nor
// the workflow define the file.
func (c *Storage) FileFullname(obj storio.Object, name string) (string, error) {
	if obj.IsOriginal(name) {
		name = obj.PrepareName(name)
		return name, nil
	}
	// Prefer the recorded meta entry (has the resolved extension).
	if item := obj.Meta().ItemByName(name); item != nil {
		return item.Fullname(), nil
	}
	// Accept the name if the workflow declares it as a target.
	if obj.Workflow().HasTarget(name) {
		return name, nil
	}
	return "", errors.Wrap(ErrCollectionFileNotDefinedInManifest, name)
}

// Clean removes all internal data from object except original
func (c *Storage) Clean(ctx context.Context, id storio.ObjectID) error {
	obj, err := c._ID2Object(ctx, id)
	if err != nil {
		return errors.Wrap(err, `Clean`)
	}
	objectpath := c.fullpath(id)
	files, err := os.ReadDir(objectpath)
	if err != nil {
		return errors.Wrap(err, `Clean`)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if file.Name() != "manifest.yaml" && !obj.IsOriginal(file.Name()) {
			fillpath := filepath.Join(objectpath, file.Name())
			if err = c.fcache.Delete(fillpath); err != nil {
				ctxlogger.Get(ctx).Error("clear cache",
					zap.String("path", fillpath),
					zap.Error(err))
			}
			if err = os.Remove(fillpath); !os.IsNotExist(err) {
				break
			}
		}
	}
	// Clean all sub-items from meta
	obj.Meta().CleanSubItems()
	return c.saveMeta(obj)
}

func (c *Storage) removeFiles(ctx context.Context, id storio.ObjectID, names ...string) error {
	var (
		obj, err   = c._ID2Object(ctx, id)
		fullname   string
		objectpath = c.fullpath(id)
	)
	if err != nil {
		return errors.Wrap(err, `remove specific files`)
	}
	for _, name := range names {
		if fullname, _ = c.FileFullname(obj, name); fullname == "" {
			fullname = name
		}
		pathname := filepath.Join(objectpath, fullname)
		if err = c.fcache.Delete(pathname); err != nil {
			ctxlogger.Get(ctx).Error("remove object cache",
				zap.String("pathname", pathname),
				zap.Error(err))
		}
		if err = os.Remove(pathname); os.IsNotExist(err) {
			if obj.MetaOrNew().Main.NameExt != "" {
				newFullname := models.SourceFilename(fullname, obj.MetaOrNew().Main.NameExt)
				newFullname = filepath.Join(objectpath, newFullname)

				if err = c.fcache.Delete(newFullname); err != nil {
					ctxlogger.Get(ctx).Error("remove object cache",
						zap.String("pathname", newFullname),
						zap.Error(err))
				}

				if err = os.Remove(newFullname); err != nil && !os.IsNotExist(err) {
					ctxlogger.Get(ctx).Error("remove sub-object",
						zap.String("pathname", newFullname),
						zap.Error(err))
				}
			}
		} else if err != nil {
			ctxlogger.Get(ctx).Error("remove sub-object base",
				zap.String("pathname", pathname),
				zap.Error(err))
		}
		// Clear "not found" errors — removing a non-existent file is benign.
		if os.IsNotExist(err) {
			err = nil
		}
		// Remove meta from file
		if obj.Meta().RemoveItemByName(fullname) {
			err = c.saveMeta(obj)
		}
		if err != nil {
			return err
		}
	}
	return err
}

// NewPath name of file
func (c *Storage) newPath(bucket string, id storio.ObjectID, check bool) (string, error) {
	if id != nil {
		if sumPath := subPathFromID(bucket, id); sumPath != "" {
			if check && !PathChecker(filepath.Join(c.root, bucket, sumPath)) {
				return "", ErrCustomObjectIDIsNotValid
			}
			return sumPath, nil
		}
	}
	return c.pathgen.Generate(filepath.Join(c.root, bucket))
}

func (c *Storage) loadWorkflow(ctx context.Context, obj storio.Object) error {
	wf, err := c.ReadWorkflow(ctx, obj.Bucket())
	if err != nil {
		return err
	}
	if wf != nil {
		*obj.WorkflowOrNew() = *wf
	}
	return nil
}

func (c *Storage) loadMeta(obj storio.Object) error {
	return loadJSONFile(c.mcache, c.subfileFullpath(obj, metaFileName), obj.MetaOrNew())
}

func (c *Storage) saveMeta(obj storio.Object) error {
	meta := obj.MetaOrNew()
	meta.UpdatedAt = time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = meta.UpdatedAt
	}
	return saveJSONFile(c.mcache, c.subfileFullpath(obj, metaFileName), meta)
}

func (c *Storage) saveObjectMeta(_ context.Context, obj storio.Object, name string, meta *models.ItemMeta) error {
	meta.UpdatedAt = time.Now()
	obj.MetaOrNew().SetItem(meta)
	return c.saveMeta(obj)
}

func (c *Storage) fullpath(id storio.ObjectID) string {
	return filepath.Join(c.root, string(id.ID()))
}

func (c *Storage) subfileFullpath(obj storio.Object, subname string) string {
	return filepath.Join(c.root, obj.Bucket(), filepath.Join(obj.Path(), subname))
}

func subPathFromID(bucket string, id storio.ObjectID) string {
	sumPath := strings.Trim(id.ID().String(), " \t\n/\\")
	if bucket != "" {
		sumPath = strings.TrimPrefix(sumPath, bucket)
		sumPath = strings.Trim(sumPath, " \t\n/\\")
	}
	return sumPath
}

// WriteFile implements ObjectFileAccessor — full implementation in Phase 3.
func (c *Storage) WriteFile(ctx context.Context, id storio.ObjectID, path string, data io.Reader, meta *models.ItemMeta) error {
	name := filepath.Join(c.objectDir(id), path)
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, data)
	return err
}

// ReadFile implements ObjectFileAccessor.
func (c *Storage) ReadFile(ctx context.Context, id storio.ObjectID, path string) (io.ReadCloser, error) {
	name := filepath.Join(c.objectDir(id), path)
	return os.Open(name)
}

// ListFiles implements ObjectFileAccessor.
func (c *Storage) ListFiles(ctx context.Context, id storio.ObjectID, pattern string) ([]*storio.FileInfo, error) {
	root := c.objectDir(id)
	var files []*storio.FileInfo
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if pattern != "" && pattern != "*" {
			if ok, _ := filepath.Match(pattern, rel); !ok {
				return nil
			}
		}
		info, _ := d.Info()
		var size int64
		if info != nil {
			size = info.Size()
		}
		files = append(files, &storio.FileInfo{Path: rel, Size: size})
		return nil
	})
	return files, err
}

// DeleteFiles implements ObjectFileAccessor.
func (c *Storage) DeleteFiles(ctx context.Context, id storio.ObjectID, paths ...string) error {
	root := c.objectDir(id)
	for _, p := range paths {
		_ = os.Remove(filepath.Join(root, p))
	}
	return nil
}

// MoveFile implements ObjectFileAccessor.
func (c *Storage) MoveFile(ctx context.Context, id storio.ObjectID, srcPath, dstPath string) error {
	root := c.objectDir(id)
	dst := filepath.Join(root, dstPath)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(filepath.Join(root, srcPath), dst)
}

// ReadState implements ObjectStateAccessor.
func (c *Storage) ReadState(ctx context.Context, id storio.ObjectID) (*models.ProcessingState, error) {
	name := filepath.Join(c.objectDir(id), "state.json")
	data, err := os.ReadFile(name)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state models.ProcessingState
	if err = json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// WriteState implements ObjectStateAccessor.
func (c *Storage) WriteState(ctx context.Context, id storio.ObjectID, state *models.ProcessingState) error {
	name := filepath.Join(c.objectDir(id), "state.json")
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(name, data, 0o644)
}

// objectDir returns the absolute directory path for an object.
func (c *Storage) objectDir(id storio.ObjectID) string {
	return filepath.Join(c.root, string(id.ID()))
}

var _ storio.StorageAccessor = (*Storage)(nil)
