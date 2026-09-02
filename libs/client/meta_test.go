package client

import (
	"testing"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/models"
	"github.com/stretchr/testify/assert"
)

func TestItemMetaFromProto_FillsNameExtAndPath(t *testing.T) {
	got := itemMetaFromProto(&protocol.ItemMeta{
		Name:        "prim",
		NameExt:     "mp4",
		Role:        "original",
		Type:        "video",
		ContentType: "video/mp4",
	})
	assert.Equal(t, "prim", got.Name)
	assert.Equal(t, "mp4", got.NameExt)
	assert.Equal(t, "prim.mp4", got.Path)
	assert.Equal(t, "original", got.Role)
	assert.Equal(t, models.TypeVideo, got.Type)
	assert.Equal(t, "prim.mp4", got.Fullname())
}

func TestItemMetaFromProto_KeepsExplicitPath(t *testing.T) {
	got := itemMetaFromProto(&protocol.ItemMeta{
		Name:    "1",
		NameExt: "jpg",
		Path:    "thumbs/1.jpg",
		Role:    "thumb",
	})
	assert.Equal(t, "thumbs/1.jpg", got.Path)
}
