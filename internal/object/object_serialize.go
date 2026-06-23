package object

import (
	"time"

	"github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

type serializeItem struct {
	ID        storio.ObjectIDType `json:"id"`
	Bucket    string              `json:"bucket"`
	Filepath  string              `json:"filepath"`
	Workflow  *models.Workflow    `json:"workflow,omitempty"`
	Meta      *models.Meta        `json:"meta"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}
