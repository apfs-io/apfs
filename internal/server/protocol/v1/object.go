package v1

import (
	"strings"
	"time"

	"github.com/apfs-io/apfs/models"
)

// NewObjectID object
func NewObjectID(id string) *ObjectID {
	return &ObjectID{Id: strings.TrimLeft(id, "/")}
}

// NewObjectGroupID object
func NewObjectGroupID(group, id string) *ObjectID {
	return &ObjectID{Id: strings.TrimLeft(group+"/"+id, "/")}
}

// NewObjectIDSubfile object
func NewObjectIDSubfile(id string, name ...string) *ObjectID {
	return &ObjectID{Id: strings.TrimLeft(id, "/"), Name: name}
}

// ToModel from protobuf object
func (m *Object) ToModel() *models.Object {
	if m == nil {
		return nil
	}
	obj := &models.Object{
		ID:     m.GetId(),
		Bucket: m.GetBucket(),
		Path:   m.GetPath(),
		HashID: m.GetHashId(),

		Status:        models.StatusFromString(m.GetStatus().GetStatus()),
		StatusMessage: m.GetStatus().GetMessage(),

		ContentType: m.GetContentType(),
		Type:        models.ObjectType(m.GetObjectType()),
		Size:        uint64(m.GetSize()),

		CreatedAt: time.Unix(0, m.GetCreatedAt()),
		UpdatedAt: time.Unix(0, m.GetUpdatedAt()),
	}
	if m.GetMeta() != nil {
		_ = obj.Meta.SetValue(m.GetMeta().ToModel())
	}
	if m.GetManifest() != nil {
		_ = obj.Manifest.SetValue(m.GetManifest().ToModel())
	}
	return obj
}

// ObjectID from the object
func (m *Object) ObjectID() *ObjectID {
	return NewObjectID(m.GetId())
}
