package object

import (
	"github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

// ToModel converts object interface to model
func ToModel(inObj storio.Object) (*models.Object, error) {
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
	if err := obj.Tags.SetValue(inObj.Meta().Tags); err != nil {
		return nil, err
	}
	if err := obj.Meta.SetValue(meta); err != nil {
		return nil, err
	}
	if wf := inObj.Workflow(); wf != nil && !wf.IsEmpty() {
		if err := obj.Workflow.SetValue(wf); err != nil {
			return nil, err
		}
	}
	return obj, nil
}

// FromModel converts model data to the object
func FromModel(obj *models.Object) storio.Object {
	meta := obj.Meta.DataOr(models.Meta{})
	wf := obj.Workflow.DataOr(models.Workflow{})
	var wfPtr *models.Workflow
	if !wf.IsEmpty() {
		wfPtr = &wf
	}
	return &Object{
		id:        storio.ObjectIDType(obj.ID),
		bucket:    obj.Bucket,
		filepath:  obj.Path,
		status:    obj.Status,
		statusMsg: obj.StatusMessage,

		meta:     &meta,
		workflow: wfPtr,

		createdAt: obj.CreatedAt,
		updatedAt: obj.UpdatedAt,
	}
}
