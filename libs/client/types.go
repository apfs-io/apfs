package client

import (
	"strings"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/models"
)

// List of alias types...
type (
	Object            = protocol.Object
	ObjectID          = protocol.ObjectID
	ObjectIDNames     = protocol.ObjectIDNames
	SimpleResponse    = protocol.SimpleResponse
	ObjectType        = models.ObjectType
	Manifest          = models.Manifest
	ManifestTaskStage = models.ManifestTaskStage
	ManifestTask      = models.ManifestTask
	Action            = models.Action
)

// PrepareObjectID ensures that the object ID includes the group prefix.
func PrepareObjectID(id *ObjectID, group string) *ObjectID {
	if !strings.HasPrefix(id.Id, group+"/") {
		return &ObjectID{
			Id:   group + "/" + id.Id,
			Name: append([]string{}, id.Name...),
		}
	}
	return id
}

// PrepareObjectIDNames ensures that the object ID includes the group prefix.
func PrepareObjectIDNames(id *ObjectIDNames, group string) *ObjectIDNames {
	if !strings.HasPrefix(id.Id, group+"/") {
		return &ObjectIDNames{
			Id:    group + "/" + id.Id,
			Names: append([]string{}, id.Names...),
		}
	}
	return id
}
