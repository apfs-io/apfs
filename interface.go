//
// @project apfs 2017 - 2022
// @author Dmitry Ponomarev <demdxx@gmail.com> 2017 - 2022
//

package apfs

import (
	"github.com/apfs-io/apfs/libs/client"
	"github.com/apfs-io/apfs/models"
)

// List of alias types.
// All types here are plain Go types — no protobuf imports at this level.
type (
	// Identity types
	ObjectID      = client.ObjectID
	ObjectIDNames = client.ObjectIDNames
	SimpleResponse = client.SimpleResponse

	// Object and metadata types exposed from the client package
	Object          = client.Object
	Meta            = client.Meta
	ItemMeta        = client.ItemMeta
	ProcessingState = client.ProcessingState
	ProcessingCounters = client.ProcessingCounters
	JobState        = client.JobState
	StepState       = client.StepState

	// Model types
	ObjectType        = models.ObjectType
	Workflow          = models.Workflow
	WorkflowJob       = models.WorkflowJob
	WorkflowStep      = models.WorkflowStep
	WorkflowValidate  = models.WorkflowValidate
	Manifest          = models.Manifest
	ManifestTaskStage = models.ManifestTaskStage
	ManifestTask      = models.ManifestTask
	Action            = models.Action
	Client            = client.Client
)

// List of constants.
const (
	OriginalFilename = models.OriginalFilename
)

// NewObjectID constructs an ObjectID from a plain ID string.
func NewObjectID(id string) *ObjectID {
	return &ObjectID{Id: id}
}

// NewObjectGroupID constructs an ObjectID scoped to the given group.
func NewObjectGroupID(group, id string) *ObjectID {
	return &ObjectID{Id: group + "/" + id}
}

// ID is a shorthand for NewObjectID.
func ID(id string) *ObjectID {
	return &ObjectID{Id: id}
}
