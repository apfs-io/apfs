package client

import (
	"time"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/models"
)

// Object is the client-facing representation of an APFS object.
// It hides internal ORM types (gosql.NullableJSON, etc.) and directly
// exposes the data that a client application needs.
//
// Workflow and State are conditionally populated: they are non-nil only
// when the corresponding RequestOption was passed (WithWorkflow, WithState,
// or WithFullState).
type Object struct {
	ID            string
	Bucket        string
	Path          string
	HashID        string
	Status        models.ObjectStatus
	StatusMessage string
	ContentType   string
	Type          models.ObjectType
	Tags          []string
	Size          uint64
	Meta          *Meta
	Workflow      *models.Workflow // non-nil only if WithWorkflow() was passed
	State         *ProcessingState // non-nil only if WithState()/WithFullState() was passed
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// objectFromProto converts the generated proto Object type to the client Object type.
// full controls whether the State's Jobs map is populated.
func objectFromProto(p *protocol.Object, full bool) *Object {
	if p == nil {
		return nil
	}
	obj := &Object{
		ID:            p.GetId(),
		Bucket:        p.GetBucket(),
		Path:          p.GetPath(),
		HashID:        p.GetHashId(),
		Status:        models.StatusFromString(p.GetStatus().GetStatus()),
		StatusMessage: p.GetStatus().GetMessage(),
		ContentType:   p.GetContentType(),
		Type:          models.ObjectType(p.GetObjectType()),
		Size:          uint64(p.GetSize()),
		Meta:          metaFromProto(p.GetMeta()),
		CreatedAt:     time.Unix(0, p.GetCreatedAt()),
		UpdatedAt:     time.Unix(0, p.GetUpdatedAt()),
	}
	if pw := p.GetWorkflow(); pw != nil {
		obj.Workflow = protocol.WorkflowToModel(pw)
	}
	if ps := p.GetState(); ps != nil {
		obj.State = stateFromProto(ps, full)
	}
	return obj
}
