package io

import (
	"time"

	"github.com/apfs-io/apfs/models"
)

// ObjectIDType contains unical number in the storage
type ObjectIDType string

func (id ObjectIDType) String() string {
	return string(id)
}

// ID of the object
func (id ObjectIDType) ID() ObjectIDType {
	return id
}

// ObjectID accessor
type ObjectID interface {
	ID() ObjectIDType
}

// ObjectStatus accessor
type ObjectStatus interface {
	Status() Status
	StatusMessage() string
	StatusUpdate(status Status)
}

// Object base info accessor
type Object interface {
	ObjectID
	ObjectStatus

	Path() string
	Bucket() string
	Revision() int64 // Shows count of changes in the object

	Meta() *models.Meta
	MustMeta() *models.Meta
	Manifest() *models.Manifest
	MustManifest() *models.Manifest

	IsOriginal(name string) bool
	PrepareName(name string) string

	CreatedAt() time.Time
	UpdatedAt() time.Time
}
