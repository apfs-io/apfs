//
// @project apfs 2017 - 2022
// @author Dmitry Ponomarev <demdxx@gmail.com> 2017 - 2022
//

package apfs

import (
	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/libs/client"
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
	Client            = client.Client
	GroupClient       = client.GroupClient
)

// List of constants...
const (
	OriginalFilename = models.OriginalFilename
)

// NewObjectID object
func NewObjectID(id string) *ObjectID {
	return protocol.NewObjectID(id)
}

// NewObjectGroupID object with group
func NewObjectGroupID(group, id string) *ObjectID {
	return protocol.NewObjectGroupID(group, id)
}
