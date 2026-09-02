package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apfs-io/apfs/models"
)

func TestMetaItemRoundTripPreservesPathRoleAttributes(t *testing.T) {
	orig := &models.ItemMeta{
		Name:        "prim",
		NameExt:     "mp4",
		Path:        "prim.mp4",
		Role:        "original",
		Type:        models.TypeVideo,
		ContentType: "video/mp4",
		Width:       1920,
		Height:      1080,
		Size:        4096,
		Attributes:  map[string]any{"codec_tag": "avc1"},
	}

	proto := MetaItemFromModel(orig)
	require.NotNil(t, proto)
	assert.Equal(t, "prim", proto.GetName())
	assert.Equal(t, "mp4", proto.GetNameExt())
	assert.Equal(t, "prim.mp4", proto.GetPath())
	assert.Equal(t, "original", proto.GetRole())
	assert.Contains(t, proto.GetAttributesJson(), "codec_tag")

	back := proto.ToModel()
	require.NotNil(t, back)
	assert.Equal(t, orig.Name, back.Name)
	assert.Equal(t, orig.NameExt, back.NameExt)
	assert.Equal(t, orig.Path, back.Path)
	assert.Equal(t, orig.Role, back.Role)
	assert.Equal(t, orig.Type, back.Type)
	assert.Equal(t, orig.ContentType, back.ContentType)
	assert.Equal(t, "avc1", back.GetAttribute("codec_tag"))
}

func TestMetaItemFromModelFillsPathFromFullname(t *testing.T) {
	proto := MetaItemFromModel(&models.ItemMeta{
		Name:    "prim",
		NameExt: "html",
	})
	require.NotNil(t, proto)
	assert.Equal(t, "prim.html", proto.GetPath())
}

func TestMetaItemToModelNil(t *testing.T) {
	var proto *ItemMeta
	assert.Nil(t, proto.ToModel())
	assert.Nil(t, MetaItemFromModel(nil))
}
