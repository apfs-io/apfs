package client

import (
	"strings"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/models"
)

// ─── Public plain-Go types ────────────────────────────────────────────────────
//
// These types are part of the client's public API and must NOT reference any
// protobuf-generated types. Internal mapping to/from protobuf happens inside
// client.go.

// ObjectID identifies an object by its full path (group/id) and an optional
// list of file-name hints (returned in preference order).
type ObjectID struct {
	Id   string
	Name []string
}

// ObjectIDNames identifies an object and a specific set of sub-item names to
// operate on.
type ObjectIDNames struct {
	Id    string
	Names []string
}

// SimpleResponse carries the outcome of a mutating API call.
type SimpleResponse struct {
	Status  string
	Message string
}

// ObjectType is a convenience alias so callers don't need to import models directly.
type ObjectType = models.ObjectType

// ─── Mapping helpers (package-private) ───────────────────────────────────────

// applyGroupPrefix prepends group/ when id is a bare object name.
// IDs that already contain a slash are treated as fully-qualified (group/path)
// and left unchanged — including when group is "default".
func applyGroupPrefix(id, group string) string {
	if group == "" || id == "" || strings.Contains(id, "/") {
		return id
	}
	return group + "/" + strings.TrimLeft(id, "/")
}

// toProtoObjectID converts a client ObjectID to a protocol ObjectID.
func toProtoObjectID(id *ObjectID, group string) *protocol.ObjectID {
	return &protocol.ObjectID{
		Id:   applyGroupPrefix(id.Id, group),
		Name: append([]string{}, id.Name...),
	}
}

// toProtoObjectIDNames converts a client ObjectIDNames to a protocol ObjectIDNames.
func toProtoObjectIDNames(id *ObjectIDNames, group string) *protocol.ObjectIDNames {
	return &protocol.ObjectIDNames{
		Id:    applyGroupPrefix(id.Id, group),
		Names: append([]string{}, id.Names...),
	}
}

// PrepareObjectID is a convenience helper for callers that construct ObjectIDs
// manually and need the group prefix applied.
func PrepareObjectID(id *ObjectID, group string) *ObjectID {
	prefixed := applyGroupPrefix(id.Id, group)
	if prefixed == id.Id {
		return id
	}
	return &ObjectID{
		Id:   prefixed,
		Name: append([]string{}, id.Name...),
	}
}

// PrepareObjectIDNames applies the group prefix to an ObjectIDNames if needed.
func PrepareObjectIDNames(id *ObjectIDNames, group string) *ObjectIDNames {
	prefixed := applyGroupPrefix(id.Id, group)
	if prefixed == id.Id {
		return id
	}
	return &ObjectIDNames{
		Id:    prefixed,
		Names: append([]string{}, id.Names...),
	}
}
