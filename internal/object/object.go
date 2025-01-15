//
// @project apfs 2018 - 2019
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2019
//

package object

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/models"
)

// Object is the main item for storing the file sets in the collection.
//
// Object can be represented like a container where stored some subfile objects
// Object also contain some additional information in `meta` and `manifest` sections.
// Meta contains information about all related to the object subfiles like thumbs or postprocessed files
type Object struct {
	// ID of the object
	id io.ObjectIDType

	// Group where located out file
	bucket string

	// filepath of root directory
	filepath string

	// manifest object information
	manifest *models.Manifest

	// meta information of files
	meta *models.Meta

	// Status of processing
	status    io.Status
	statusMsg string

	createdAt time.Time
	updatedAt time.Time
}

// NewObject liked to bucket and object codename
func NewObject(id io.ObjectIDType, bucket, filepath string) *Object {
	return &Object{id: id, bucket: bucket, filepath: filepath}
}

func (f *Object) String() string {
	return f.ID().String()
}

// ID of the object in the storage
func (f *Object) ID() io.ObjectIDType {
	return f.id
}

// Bucket name where stored the object
// Like: videos, images, documents, etc.
func (f *Object) Bucket() string {
	return f.bucket
}

// Path returns unical name of the object in the paticular `bucket`
func (f *Object) Path() string {
	return f.filepath
}

// Revision shows count of changes in the object
func (f *Object) Revision() int64 {
	return 0
}

// Meta information of the object
func (f *Object) Meta() *models.Meta {
	return f.meta
}

// MustMeta information returns from the object or creates new if not exists
func (f *Object) MustMeta() *models.Meta {
	if f.meta == nil {
		f.meta = &models.Meta{}
	}
	return f.meta
}

// Manifest information of the object
func (f *Object) Manifest() *models.Manifest {
	return f.manifest
}

// MustManifest information returns from the object or creates new if not exists
func (f *Object) MustManifest() *models.Manifest {
	if f.manifest == nil {
		f.manifest = &models.Manifest{}
	}
	return f.manifest
}

// Status returns the object status
func (f *Object) Status() io.Status {
	return f.status
}

// StatusMessage returns common message of all items
func (f *Object) StatusMessage() string {
	var msgs []string
	for _, task := range f.meta.Tasks {
		if task.StatusMessage != "" {
			msgs = append(msgs, fmt.Sprintf("%s: %s", task.ID, task.StatusMessage))
		}
	}
	return strings.Join(msgs, "\n")
}

// StatusUpdate state
func (f *Object) StatusUpdate(status io.Status) {
	f.status = status
}

// PrepareName of the subfile
func (f *Object) PrepareName(name string) string {
	if models.IsOriginal(name) {
		if f.meta != nil {
			if f.meta.Main.NameExt != "" {
				name = models.OriginalFilename + "." + f.meta.Main.NameExt
			} else {
				name = f.meta.Main.Name
			}
		}
		if name == `` {
			name = models.OriginalFilename
		}
	}
	return name
}

// IsOriginal filename
func (f *Object) IsOriginal(name string) bool {
	return models.IsOriginal(name)
}

// ToModel object
func (f *Object) ToModel() (*models.Object, error) {
	var (
		meta = f.meta
		obj  = &models.Object{
			Path:          f.filepath,
			Status:        f.status,
			StatusMessage: f.StatusMessage(),
			HashID:        meta.Main.HashID,
			ContentType:   meta.Main.ContentType,
			Type:          meta.Main.Type,
			Size:          uint64(meta.Main.Size),
		}
	)
	if err := obj.Tags.SetValue(meta.Tags); err != nil {
		return nil, err
	}
	if err := obj.Meta.SetValue(meta); err != nil {
		return nil, err
	}
	return obj, nil
}

// MarshalJSON implements method of json.Marshaler
func (f *Object) MarshalJSON() ([]byte, error) {
	return json.Marshal(&serializeItem{
		ID:        f.id,
		Bucket:    f.bucket,
		Filepath:  f.filepath,
		Manifest:  f.manifest,
		Meta:      f.meta,
		CreatedAt: f.createdAt,
		UpdatedAt: f.updatedAt,
	})
}

// UnmarshalJSON implements method of json.Unmarshaler
func (f *Object) UnmarshalJSON(data []byte) error {
	var (
		objState serializeItem
		err      = json.Unmarshal(data, &objState)
	)
	if err == nil {
		*f = Object{
			id:        objState.ID,
			bucket:    objState.Bucket,
			filepath:  objState.Filepath,
			manifest:  objState.Manifest,
			meta:      objState.Meta,
			createdAt: objState.CreatedAt,
			updatedAt: objState.UpdatedAt,
		}
	}
	return err
}

// CreatedAt returns time of the object
func (f *Object) CreatedAt() time.Time {
	return f.createdAt
}

// UpdatedAt returns time of the object
func (f *Object) UpdatedAt() time.Time {
	return f.updatedAt
}

var (
	_ json.Marshaler   = (*Object)(nil)
	_ json.Unmarshaler = (*Object)(nil)
)
