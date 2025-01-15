package object

import (
	"github.com/geniusrabbit/gosql/v2"

	"github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/models"
)

// ToModel converts object interface to model
func ToModel(inObj io.Object) (*models.Object, error) {
	meta := inObj.Meta()
	obj := &models.Object{
		ID:            inObj.ID().String(),
		Bucket:        inObj.Bucket(),
		Path:          inObj.Path(),
		Status:        inObj.Status(),
		StatusMessage: inObj.StatusMessage(),
		HashID:        meta.Main.HashID,
		ContentType:   meta.Main.ContentType,
		Type:          meta.Main.Type,
		Size:          uint64(meta.Main.Size),
	}
	tags := gosql.NullableStringArray(inObj.Meta().Tags)
	if err := obj.Tags.SetValue(tags); err != nil {
		return nil, err
	}
	if err := obj.Meta.SetValue(meta); err != nil {
		return nil, err
	}
	if err := obj.Manifest.SetValue(inObj.Manifest()); err != nil {
		return nil, err
	}
	return obj, nil
}

// FromModel converts model data to the object
func FromModel(obj *models.Object) io.Object {
	meta := obj.Meta.DataOr(models.Meta{})
	manifest := obj.Manifest.DataOr(models.Manifest{})
	manifestPtr := &manifest
	if manifest.IsEmpty() {
		manifestPtr = nil
	}
	outObj := &Object{
		id:        io.ObjectIDType(obj.ID),
		bucket:    obj.Bucket,
		filepath:  obj.Path,
		status:    obj.Status,
		statusMsg: obj.StatusMessage,

		meta:     &meta,
		manifest: manifestPtr,

		createdAt: obj.CreatedAt,
		updatedAt: obj.UpdatedAt,
	}
	return outObj
}
