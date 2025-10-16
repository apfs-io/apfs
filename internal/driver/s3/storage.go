// Package s3 implements file storage collection with S3 driver support
package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/io/objectpath"
	datalib "github.com/apfs-io/apfs/internal/storage/data"
	"github.com/apfs-io/apfs/internal/utils"
	"github.com/apfs-io/apfs/libs/storerrors"
	"github.com/apfs-io/apfs/models"
)

const (
	manifestObjectName = "manifest.json"
	metaObjectName     = "meta.json"
)

// Errors list...
var (
	ErrUnsupportedContentType   = errors.New("content-type is not supported")
	ErrCustomObjectIDIsNotValid = errors.New("invalid custom object ID or taken")
)

// Storage to manage S3 type
type Storage struct {
	mx sync.RWMutex

	pathgen objectpath.Generator

	// Connection to the S3 server
	c *awss3.Client

	// Main bucket name.
	// If this parameter was setup then all objects
	// will be stored into the subdirectory
	bucketName string

	// List of used buckets for using
	realBuckets map[string]bool
}

// NewStorage object returns storage according to options
func NewStorage(ctx context.Context, options ...Options) (*Storage, error) {
	optConfig := optionConfig{
		config: aws.NewConfig(),
	}

	for _, opt := range options {
		if err := opt(&optConfig); err != nil {
			return nil, err
		}
	}
	if len(optConfig.s3confOptions) == 0 {
		optConfig.s3confOptions = append(optConfig.s3confOptions, func(o *awss3.Options) {
			o.UsePathStyle = true
		})
	}

	store := &Storage{
		c:           awss3.NewFromConfig(*optConfig.config, optConfig.s3confOptions...),
		bucketName:  optConfig.mainBucketName,
		realBuckets: map[string]bool{},
	}
	store.pathgen = optConfig._pathgen(func(path string) bool {
		return store.isValidObjectPath(ctx, path)
	})
	if store.bucketName != "" {
		if err := store.createBucketIfNotExists(ctx, store.bucketName); err != nil {
			return nil, err
		}
	}
	return store, nil
}

// ReadManifest information method
func (c *Storage) ReadManifest(ctx context.Context, bucket string) (*models.Manifest, error) {
	var (
		object = newObject(bucket, ".dummy")
		err    = c.loadManifest(ctx, object, true)
	)
	if isNotExist(err) {
		return nil, storerrors.WrapNotFound(object.Path(), err)
	}
	return object.Manifest(), err
}

// UpdateManifest information method
func (c *Storage) UpdateManifest(ctx context.Context, bucket string, manifest *models.Manifest) error {
	if manifest == nil {
		return ErrInvalidBucketManifest
	}
	object := newObject(bucket, ".dummy")
	*object.MustManifest() = *manifest
	return c.saveManifest(ctx, object, true)
}

// Create new file object
func (c *Storage) Create(ctx context.Context, bucket string, id npio.ObjectID, overwrite bool, params url.Values) (npio.Object, error) {
	objectName, err := c.newObjectName(ctx, bucket, id)
	if err != nil {
		return nil, err
	}

	// Init new object container
	object := newObject(bucket, objectName)

	// Load manifest information
	if err = c.loadManifest(ctx, object, true); err != nil {
		if !isNotExist(err) {
			return nil, err
		}
	}

	// Update meta tags information
	if params != nil && len(params["tags"]) > 0 {
		object.MustMeta().Tags = params["tags"]
		if err = c.saveMeta(ctx, object); err != nil {
			return nil, err
		}
	}
	return object, err
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
	return c.saveMeta(ctx, obj)
}

// Open existing file
func (c *Storage) Open(ctx context.Context, id npio.ObjectID) (_ npio.Object, err error) {
	// Init object container by ID
	object := objectFromID(id)

	// Load object meta information
	if err = c.loadMeta(ctx, object); err != nil {
		if isNotExist(err) {
			return nil, storerrors.WrapNotFound(object.Path(), err)
		}
	}

	// Load object manifest information
	if err = c.loadObjectManifest(ctx, object); err != nil {
		if !isNotExist(err) {
			return nil, err
		}
		err = nil
	}

	// Remove all subfiles from object if no main file is present
	meta := object.Meta()
	if (len(meta.Tasks) > 0 || len(meta.Items) > 0) && meta.Main.IsEmpty() {
		if err = c.Clean(ctx, object); err != nil {
			return nil, err
		}
	}
	return object, err
}

// Upload data as file
func (c *Storage) Update(ctx context.Context, id npio.ObjectID, name string, reader io.Reader, meta *models.ItemMeta) error {
	// Get object by ID
	object, err := c._ID2Object(ctx, id)
	if err != nil {
		return err
	}

	// Update only metainformation of the object
	if reader == nil {
		return c.saveObjectMeta(ctx, name, object, meta)
	}

	// Extract meta information from the reader
	data, meta, err := c.extractObjectMetaInfo(name, reader, object.Manifest(), meta)
	if err != nil {
		return err
	}

	// Update meta information
	object.MustMeta().SetItem(meta)

	// Save data to the S3 server
	return c.putData(ctx, meta.Fullname(), object, data, meta, false)
}

// Update data in the storage
func (c *Storage) UpdateMeta(ctx context.Context, id npio.ObjectID, name string, meta *models.ItemMeta) error {
	obj, err := c._ID2Object(ctx, id)
	if err != nil {
		return err
	}
	return c.saveObjectMeta(ctx, name, obj, meta)
}

// Remove file from directory by name without extension of file
func (c *Storage) Remove(ctx context.Context, id npio.ObjectID, names ...string) error {
	object, err := c._ID2Object(ctx, id)
	if err != nil {
		return err
	}

	// Get object bucket name
	bucket := object.Bucket()

	// List of names to remove
	if len(names) < 1 {
		objects, err := c.listOfObjects(ctx, bucket, objectKey(object, ""))
		if err != nil {
			return err
		}
		for _, obj := range objects {
			names = append(names, *obj.Key)
		}
	}

	// Meta update flag
	metaUpdated := false

	// Remove all subfiles from object
	for _, name := range names {
		_, err = c.c.DeleteObject(ctx, &awss3.DeleteObjectInput{
			Bucket: c._bucketName(bucket),
			Key:    aws.String(name),
		})

		if err == nil {
			baseName, updated := filepath.Base(name), false
			// Remove meta from file
			if updated, err = object.Meta().RemoveItemByName(baseName); updated {
				metaUpdated = true
			}
		}

		// Check if the object is not exist or error
		if err != nil && !isNoSuchKeyError(err) {
			if metaUpdated {
				_ = c.saveMeta(ctx, object)
			}
			return err
		}
	}

	// Save meta information if needed
	if metaUpdated {
		return c.saveMeta(ctx, object)
	}
	return nil
}

// Read returns reader of the specific internal object
func (c *Storage) Read(ctx context.Context, id npio.ObjectID, name string) (io.ReadCloser, error) {
	object, err := c._ID2Object(ctx, id)
	if err != nil {
		return nil, err
	}
	reader, err := c.loadData(ctx, object, name, false)
	if isNotExist(err) {
		return nil, storerrors.WrapNotFound(name, err)
	}
	return reader, err
}

// Clean removes all subfiles from object except original and manifest
func (c *Storage) Clean(ctx context.Context, id npio.ObjectID) error {
	object, objErr := c._ID2Object(ctx, id)
	if objErr != nil {
		return objErr
	}
	var (
		bucket       = object.Bucket()
		objects, err = c.listOfObjects(ctx, bucket, object.Path())
		objectPath   = *c._bucketFilenameBasic(bucket, object.Path())
	)
	if err != nil {
		return err
	}
	for _, obj := range objects {
		baseName := strings.TrimPrefix(strings.TrimPrefix(*obj.Key, objectPath), `/`)
		if baseName == manifestObjectName {
			continue
		}
		if object.IsOriginal(baseName) {
			if object.MustMeta().Main.IsEmpty() {
				if err = c.refreshObjectMainMeta(ctx, object, *obj.Key); err != nil {
					return err
				}
			}
			continue
		}
		_, err = c.c.DeleteObject(ctx, &awss3.DeleteObjectInput{
			Bucket: c._bucketName(bucket),
			Key:    &[]string{*obj.Key}[0],
		})
		if err != nil {
			return err
		}
	}
	// Clean all sub-items from meta
	object.MustMeta().CleanSubItems()
	return c.saveMeta(ctx, object)
}

// Scan storage by pattern
//
//	pattern: search type equals to glob https://golang.org/pkg/path/filepath/#Glob
func (c *Storage) Scan(ctx context.Context, pattern string, walkf npio.WalkStorageFnk) error {
	pattern = strings.TrimLeft(pattern, "/")
	arr := strings.SplitN(pattern, "/", 2)
	bucket, prefix := arr[0], arr[1]
	output, err := c.c.ListObjects(ctx, &awss3.ListObjectsInput{
		Bucket: c._bucketName(bucket),
		Prefix: c._bucketFilenameBasic(bucket, prefix),
	})
	if err != nil {
		return err
	}
	for _, o := range output.Contents {
		name := c._keyName(ctx, *o.Key)
		if ok, _ := filepath.Match(pattern, name); ok {
			if err = walkf(name, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Storage) _ID2Object(ctx context.Context, id npio.ObjectID) (npio.Object, error) {
	switch obj := id.(type) {
	case npio.Object:
		return obj, nil
	default:
		return c.Open(ctx, id)
	}
}

// Create the bucket if it doesn't exist yet.
func (c *Storage) createBucketIfNotExists(ctx context.Context, bucketName string) error {
	if c.isBucketCreated(bucketName) {
		return nil
	}

	c.mx.Lock()
	defer c.mx.Unlock()

	listBucketsOutput, err := c.c.ListBuckets(ctx, &awss3.ListBucketsInput{})
	if err != nil {
		return err
	}

	ownsBucket := slices.ContainsFunc(listBucketsOutput.Buckets,
		func(b awss3types.Bucket) bool { return *b.Name == bucketName })

	if !ownsBucket {
		_, err = c.c.CreateBucket(ctx, &awss3.CreateBucketInput{
			Bucket: aws.String(bucketName),
		})
	}

	if err == nil {
		c.realBuckets[bucketName] = true
	}
	return nil
}

func (c *Storage) isBucketCreated(bucketName string) bool {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.realBuckets[bucketName]
}

// newObjectName returns the object codename
func (c *Storage) newObjectName(ctx context.Context, bucket string, id npio.ObjectID) (string, error) {
	if id != nil {
		if sumPath := subPathFromID(bucket, id); sumPath != "" {
			if !c.isValidObjectPath(ctx, filepath.Join(bucket, sumPath)) {
				return "", errors.Wrap(ErrCustomObjectIDIsNotValid, sumPath)
			}
			return sumPath, nil
		}
	}
	return c.pathgen.Generate(bucket)
}

func (c *Storage) loadObjectManifest(ctx context.Context, object npio.Object) (err error) {
	if err = c.loadManifest(ctx, object, false); err == nil {
		return nil
	}
	if isNotExist(err) {
		err = c.loadManifest(ctx, object, true)
	}
	if err != nil && !isNotExist(err) {
		err = errors.Wrap(err, manifestObjectName)
	} else {
		err = nil
	}
	return err
}

func (c *Storage) loadManifest(ctx context.Context, object npio.Object, global bool) error {
	return c.loadJSONObject(ctx, object, manifestObjectName, object.MustManifest(), global)
}

func (c *Storage) saveManifest(ctx context.Context, object npio.Object, global bool) error {
	return c.putJSONObject(ctx, manifestObjectName, object, object.MustManifest(), global)
}

func (c *Storage) loadMeta(ctx context.Context, object npio.Object) error {
	meta := object.MustMeta()
	err := c.loadJSONObject(ctx, object, metaObjectName, meta, false)
	if err == nil {
		if meta.CreatedAt.IsZero() {
			meta.CreatedAt = time.Now()
		}
		if meta.UpdatedAt.IsZero() {
			meta.UpdatedAt = meta.CreatedAt
		}
	}
	return err
}

func (c *Storage) saveMeta(ctx context.Context, object npio.Object) error {
	meta := object.MustMeta()
	meta.UpdatedAt = time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = meta.UpdatedAt
	}
	return c.putJSONObject(ctx, metaObjectName, object, meta, false)
}

func (c *Storage) putJSONObject(ctx context.Context, name string, object npio.Object, item any, global bool) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return c.putData(ctx, name, object, bytes.NewReader(data),
		&models.ItemMeta{ContentType: "application/json"}, global)
}

func (c *Storage) putData(ctx context.Context, name string, object npio.Object, data io.ReadSeeker, meta *models.ItemMeta, global bool) (err error) {
	// Prepare object input object
	pubObjectInput := awss3.PutObjectInput{
		Body:   data,
		Bucket: c._bucketName(object.Bucket()),
		Key:    c._bucketFilename(object, strIf(global, name, objectKey(object, name))),
	}

	// Add tags to the object
	if len(object.MustMeta().Tags) != 0 {
		tags := url.Values{}
		for _, tag := range object.MustMeta().Tags {
			tags.Set(tag, "1")
		}
		pubObjectInput.Tagging = aws.String(tags.Encode())
	}

	// Set content type
	if meta != nil {
		pubObjectInput.ContentType = aws.String(meta.ContentType)
	}

	// Set permissions for the object in S3
	c.grantPermissions(name, object, &pubObjectInput)

	// Save data to the S3 server
	if _, err = c.c.PutObject(ctx, &pubObjectInput); err != nil {
		return err
	}

	// Save meta information if needed
	if meta != nil && name != metaObjectName && name != manifestObjectName {
		err = c.saveObjectMeta(ctx, name, object, meta)
	}
	return err
}

func (c *Storage) saveObjectMeta(ctx context.Context, _ string, object npio.Object, meta *models.ItemMeta) error {
	meta.UpdatedAt = time.Now()
	object.MustMeta().SetItem(meta)
	return c.saveMeta(ctx, object)
}

func (c *Storage) loadData(ctx context.Context, object npio.Object, name string, global bool) (io.ReadCloser, error) {
	out, err := c.c.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: c._bucketName(object.Bucket()),
		Key:    c._bucketFilename(object, strIf(global, name, objectKey(object, name))),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, err
}

func (c *Storage) loadJSONObject(ctx context.Context, object npio.Object, name string, target any, global bool) error {
	data, err := c.loadData(ctx, object, name, global)
	if err != nil {
		return err
	}
	defer func() { _ = data.Close() }()
	return json.NewDecoder(data).Decode(target)
}

func (c *Storage) _bucketName(name string) *string {
	if len(c.bucketName) > 0 {
		return &c.bucketName
	}
	name = strings.TrimLeft(name, "/")
	return &name
}

func (c *Storage) _keyName(ctx context.Context, name string) string {
	name = strings.TrimLeft(name, "/")
	if len(c.bucketName) > 0 {
		if strings.HasPrefix(name, c.bucketName) {
			name = strings.TrimLeft(strings.TrimPrefix(name, c.bucketName), "/")
		} else {
			ctxlogger.Get(ctx).Debug("invalid key to scan", zap.String("key", name))
		}
	}
	return name
}

// _bucketFilename returns path inside the bucket or
func (c *Storage) _bucketFilename(object npio.Object, name string) *string {
	return c._bucketFilenameBasic(object.Bucket(), object.PrepareName(name))
}

// In case if we have global bucket the `bucketName` parameter applies as a directory prefix
func (c *Storage) _bucketFilenameBasic(bucketName, name string) *string {
	if len(c.bucketName) > 0 {
		name = bucketName + "/" + name
		return &name
	}
	return &name
}

func (c *Storage) listOfObjects(ctx context.Context, bucket, prefix string) ([]awss3types.Object, error) {
	output, err := c.c.ListObjects(ctx, &awss3.ListObjectsInput{
		Bucket: c._bucketName(bucket),
		Prefix: c._bucketFilenameBasic(bucket, prefix),
	})
	if err != nil {
		return nil, err
	}
	return output.Contents, nil
}

func (c *Storage) grantPermissions(name string, _ npio.Object, obj *awss3.PutObjectInput) {
	switch name {
	case metaObjectName, manifestObjectName:
		obj.ACL = awss3types.ObjectCannedACLPrivate
	default:
		obj.ACL = awss3types.ObjectCannedACLPublicRead
	}
}

// the method updates object meta for the main (original) object file
func (c *Storage) refreshObjectMainMeta(ctx context.Context, object npio.Object, name string) error {
	objectPath := *c._bucketFilenameBasic(object.Bucket(), object.Path())
	baseName := strings.TrimPrefix(strings.TrimPrefix(name, objectPath), `/`)

	// Update main file info if empty
	reader, err := c.loadData(ctx, object, baseName, false)
	if err != nil {
		return err
	}
	transformReader, _, err := c.extractObjectMetaInfo(name, reader,
		object.Manifest(), &object.MustMeta().Main)
	if err != nil {
		_ = closeObject(reader)
		return err
	}
	return closeObject(transformReader)
}

func (c *Storage) extractObjectMetaInfo(name string, reader io.Reader, manifest *models.Manifest, meta *models.ItemMeta) (io.ReadSeeker, *models.ItemMeta, error) {
	var (
		data, err   = datalib.ToReadSeeker(reader)
		contentType string
	)
	if err != nil {
		return nil, nil, err
	}

	if contentType, err = datalib.ContentTypeByReadSeeker(data); err != nil {
		return nil, nil, err
	}

	if models.IsOriginal(name) && !manifest.IsValidContentType(contentType) {
		return nil, nil, errors.Wrap(ErrUnsupportedContentType, contentType)
	}

	if filepath.Ext(name) == "" {
		name += datalib.ExtensionByContentType(contentType)
	}

	if meta == nil {
		meta = &models.ItemMeta{}
	}

	meta.ContentType = contentType
	if meta, err = utils.CollectReadSeekerInfo(meta, data, name, ""); err != nil {
		return nil, nil, err
	}
	return data, meta, err
}

func (c *Storage) isValidObjectPath(ctx context.Context, fullpath string) bool {
	paths := strings.SplitN(fullpath, "/", 2)
	if len(paths) != 2 {
		return false
	}
	objects, err := c.listOfObjects(ctx, paths[0], paths[1])
	if isNotExist(err) {
		return true
	}
	return err == nil && len(objects) == 0
}

func subPathFromID(bucket string, id npio.ObjectID) string {
	sumPath := strings.Trim(id.ID().String(), " \t\n/\\")
	if bucket != "" {
		sumPath = strings.TrimPrefix(sumPath, bucket)
		sumPath = strings.Trim(sumPath, " \t\n/\\")
	}
	return sumPath
}

// isNoSuchKeyError checks if the error is an AWS S3 NoSuchKey error
func isNoSuchKeyError(err error) bool {
	var ae interface{ ErrorCode() string }
	if err == nil {
		return false
	}
	if errors.As(err, &ae) {
		return ae.ErrorCode() == "NoSuchKey"
	}
	return strings.Contains(err.Error(), "NoSuchKey")
}

var _ npio.StorageAccessor = (*Storage)(nil)
