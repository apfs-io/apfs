package object

import (
	"time"

	storio "github.com/apfs-io/apfs/internal/storio"
)

// Timestamps returns the best-known created/updated times for an object.
// When the object wrapper has zero values (common for S3-backed objects),
// timestamps are taken from meta.json.
func Timestamps(obj storio.Object) (created, updated time.Time) {
	created = obj.CreatedAt()
	updated = obj.UpdatedAt()
	if meta := obj.Meta(); meta != nil {
		if created.IsZero() && !meta.CreatedAt.IsZero() {
			created = meta.CreatedAt
		}
		if updated.IsZero() && !meta.UpdatedAt.IsZero() {
			updated = meta.UpdatedAt
		}
	}
	if updated.IsZero() {
		updated = created
	}
	if created.IsZero() {
		created = updated
	}
	return created, updated
}

// TouchUpdatedAt sets the wrapper updated-at timestamp.
func TouchUpdatedAt(obj storio.Object, t time.Time) {
	if o, ok := obj.(*Object); ok {
		o.SetUpdatedAt(t)
	}
}
