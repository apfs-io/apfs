// Package fs implements file storage collection on file (disk) system
package fs

import (
	"context"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/io/objectpath"
	"github.com/apfs-io/apfs/internal/object"
	datalib "github.com/apfs-io/apfs/internal/storage/data"
	"github.com/apfs-io/apfs/internal/utils"
	"github.com/apfs-io/apfs/libs/storerrors"
	"github.com/apfs-io/apfs/models"
)

const (
	manifestFileName = "manifest.json"
	metaFileName     = "meta.json"
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

// ReadManifest by default for the BUCKET
func (c *Storage) ReadManifest(ctx context.Context, bucket string) (*models.Manifest, error) {
	var (
		manifest models.Manifest
		err      = loadJSONFile(c.mcache, filepath.Join(c.root, bucket, manifestFileName), &manifest)
	)
	if err != nil {
		return nil, err
	}
	return &manifest, err
}

// UpdateManifest by default for the BUCKET
func (c *Storage) UpdateManifest(ctx context.Context, bucket string, manifest *models.Manifest) error {
	return saveJSONFile(c.mcache, filepath.Join(c.root, bucket, manifestFileName), manifest)
}

// Create new file object
func (c *Storage) Create(ctx context.Context, bucket string, id npio.ObjectID, overwrite bool, params url.Values) (npio.Object, error) {
	ctxlogger.Get(ctx).Info("Create", zap.Any("id", id))
	var (
		path, err = c.newPath(bucket, id, !overwrite)
		info      os.FileInfo
	)
	if err != nil {
		return nil, err
	}

	// Init new object container
	obj := object.NewObject(
		npio.ObjectIDType(filepath.Join(bucket, path)),
		bucket, path)

	if overwrite {
		if err := c.Remove(ctx, obj); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Load manifest information
	if err = c.loadManifest(obj); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	// Update meta information of the object
	if len(params) > 0 {
		tags := params["tags"]
		params.Del("tags")
		meta := obj.MustMeta()
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
	updateObjectFileInfo(obj, info)

	return obj, err
}

// UpdatePatams in the object. If name is present then update only params linked with the subobject
func (c *Storage) UpdatePatams(ctx context.Context, id npio.ObjectID, params url.Values) error {
	obj, err := c._ID2Object(ctx, id)
	if err != nil {
		return err
	}
	meta := obj.MustMeta()
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
func (c *Storage) Scan(ctx context.Context, pattern string, walkf npio.WalkStorageFnk) error {
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
func (c *Storage) Open(ctx context.Context, id npio.ObjectID) (_ npio.Object, err error) {
	var (
		info     os.FileInfo
		object   = objectFromID(id)
		fullpath = c.fullpath(object)
	)
	if info, err = os.Stat(fullpath); os.IsNotExist(err) {
		return nil, storerrors.WrapNotFound(fullpath, err)
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrCollectionFilePathMustBeDirectory
	}
	if err = c.loadManifest(object); err != nil {
		if err == ErrCollectionFileOriginalFilepath {
			return nil, err
		}
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, manifestFileName)
		}
	}
	if err = c.loadMeta(object); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	updateObjectFileInfo(object, info)
	return object, nil
}

func (c *Storage) _ID2Object(ctx context.Context, id npio.ObjectID) (npio.Object, error) {
	switch t := id.(type) {
	case npio.Object:
		return t, nil
	default:
		return c.Open(ctx, id)
	}
}

// Update data in the storage
func (c *Storage) Update(ctx context.Context, id npio.ObjectID, name string, data io.Reader, meta *models.ItemMeta) error {
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

	if !obj.Manifest().IsValidContentType(contentType) {
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
func (c *Storage) UpdateMeta(ctx context.Context, id npio.ObjectID, name string, meta *models.ItemMeta) error {
	obj, err := c._ID2Object(ctx, id)
	if err != nil {
		return err
	}
	return c.saveObjectMeta(ctx, obj, name, meta)
}

// Remove file from directory by name without extension of file
func (c *Storage) Remove(ctx context.Context, id npio.ObjectID, names ...string) error {
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
func (c *Storage) Read(ctx context.Context, id npio.ObjectID, name string) (io.ReadCloser, error) {
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

// FileFullname with extension
func (c *Storage) FileFullname(obj npio.Object, name string) (string, error) {
	if obj.IsOriginal(name) {
		name = obj.PrepareName(name)
	} else if task := c.filetaskByManifest(obj, name); task == nil {
		return "", errors.Wrap(ErrCollectionFileNotDefinedInManifest, name)
	} else if item := obj.Meta().ItemByName(name); item != nil {
		name = item.Fullname()
	}
	return name, nil
}

// Clean removes all internal data from object except original
func (c *Storage) Clean(ctx context.Context, id npio.ObjectID) error {
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
		if file.Name() != manifestFileName && !obj.IsOriginal(file.Name()) {
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

func (c *Storage) removeFiles(ctx context.Context, id npio.ObjectID, names ...string) error {
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
			if obj.MustMeta().Main.NameExt != "" {
				newFullname := models.SourceFilename(fullname, obj.MustMeta().Main.NameExt)
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
		// Remove meta from file
		updated := false
		if updated, err = obj.Meta().RemoveItemByName(fullname); updated && err == nil {
			err = c.saveMeta(obj)
		}
		if err != nil {
			return err
		}
	}
	return err
}

// NewPath name of file
func (c *Storage) newPath(bucket string, id npio.ObjectID, check bool) (string, error) {
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

func (c *Storage) filetaskByManifest(obj npio.Object, name string) *models.ManifestTask {
	return obj.Manifest().TaskByTarget(name)
}

func (c *Storage) loadManifest(obj npio.Object) error {
	return loadJSONFile(
		c.mcache,
		filepath.Join(c.root, obj.Bucket(), manifestFileName),
		obj.MustManifest())
}

func (c *Storage) loadMeta(obj npio.Object) error {
	return loadJSONFile(c.mcache, c.subfileFullpath(obj, metaFileName), obj.MustMeta())
}

func (c *Storage) saveMeta(obj npio.Object) error {
	meta := obj.MustMeta()
	meta.UpdatedAt = time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = meta.UpdatedAt
	}
	return saveJSONFile(c.mcache, c.subfileFullpath(obj, metaFileName), meta)
}

func (c *Storage) saveObjectMeta(ctx context.Context, obj npio.Object, name string, meta *models.ItemMeta) error {
	// if models.IsOriginal(name) {
	// 	obj.MustMeta().Main = *meta
	// } else {
	meta.UpdatedAt = time.Now()
	obj.MustMeta().SetItem(meta)
	// }
	return c.saveMeta(obj)
}

func (c *Storage) fullpath(id npio.ObjectID) string {
	return filepath.Join(c.root, string(id.ID()))
}

func (c *Storage) subfileFullpath(obj npio.Object, subname string) string {
	return filepath.Join(c.root, obj.Bucket(), filepath.Join(obj.Path(), subname))
}

func subPathFromID(bucket string, id npio.ObjectID) string {
	sumPath := strings.Trim(id.ID().String(), " \t\n/\\")
	if bucket != "" {
		sumPath = strings.TrimPrefix(sumPath, bucket)
		sumPath = strings.Trim(sumPath, " \t\n/\\")
	}
	return sumPath
}

var _ npio.StorageAccessor = (*Storage)(nil)
