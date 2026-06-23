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

// Model aliases shared with models — keep as type aliases so callers don't need
// two separate imports.
type (
	ObjectType        = models.ObjectType
	Manifest          = models.Manifest
	ManifestTaskStage = models.ManifestTaskStage
	ManifestTask      = models.ManifestTask
	Action            = models.Action
)

// ─── Mapping helpers (package-private) ───────────────────────────────────────

// toProtoObjectID converts a client ObjectID to a protocol ObjectID.
func toProtoObjectID(id *ObjectID, group string) *protocol.ObjectID {
	fullID := id.Id
	if group != "" && !strings.HasPrefix(fullID, group+"/") {
		fullID = group + "/" + strings.TrimLeft(fullID, "/")
	}
	return &protocol.ObjectID{
		Id:   fullID,
		Name: append([]string{}, id.Name...),
	}
}

// toProtoObjectIDNames converts a client ObjectIDNames to a protocol ObjectIDNames.
func toProtoObjectIDNames(id *ObjectIDNames, group string) *protocol.ObjectIDNames {
	fullID := id.Id
	if group != "" && !strings.HasPrefix(fullID, group+"/") {
		fullID = group + "/" + strings.TrimLeft(fullID, "/")
	}
	return &protocol.ObjectIDNames{
		Id:    fullID,
		Names: append([]string{}, id.Names...),
	}
}

// PrepareObjectID is a convenience helper for callers that construct ObjectIDs
// manually and need the group prefix applied.
func PrepareObjectID(id *ObjectID, group string) *ObjectID {
	if group != "" && !strings.HasPrefix(id.Id, group+"/") {
		return &ObjectID{
			Id:   group + "/" + id.Id,
			Name: append([]string{}, id.Name...),
		}
	}
	return id
}

// PrepareObjectIDNames applies the group prefix to an ObjectIDNames if needed.
func PrepareObjectIDNames(id *ObjectIDNames, group string) *ObjectIDNames {
	if group != "" && !strings.HasPrefix(id.Id, group+"/") {
		return &ObjectIDNames{
			Id:    group + "/" + id.Id,
			Names: append([]string{}, id.Names...),
		}
	}
	return id
}
