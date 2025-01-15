package object

import (
	"time"

	"github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/models"
)

type serializeItem struct {
	ID        io.ObjectIDType  `json:"id"`
	Bucket    string           `json:"bucket"`
	Filepath  string           `json:"filepath"`
	Manifest  *models.Manifest `json:"manifest"`
	Meta      *models.Meta     `json:"meta"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}
