package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetaItemByName(t *testing.T) {
	meta := Meta{
		Main: ItemMeta{Name: "original", NameExt: "mp4", Path: "original.mp4"},
		Items: []*ItemMeta{
			{Name: "main", NameExt: "mp4", Path: "main.mp4"},
			{Name: "1", NameExt: "jpg", Path: "thumbs/1.jpg"},
		},
	}

	assert.Equal(t, "original.mp4", meta.ItemByName("@").Path)
	assert.Equal(t, "original.mp4", meta.ItemByName("").Path)
	assert.Equal(t, "main.mp4", meta.ItemByName("main").Path)
	assert.Equal(t, "thumbs/1.jpg", meta.ItemByName("thumbs/1.jpg").Path)
}

func TestMetaSetItem(t *testing.T) {
	meta := Meta{}
	item := &ItemMeta{Name: "preview", NameExt: "jpg", Path: "preview.jpg"}
	meta.SetItem(item)
	assert.Equal(t, 1, len(meta.Items))

	// upsert
	updated := &ItemMeta{Name: "preview", NameExt: "jpg", Path: "preview.jpg", Size: 1024}
	meta.SetItem(updated)
	assert.Equal(t, 1, len(meta.Items))
	assert.Equal(t, int64(1024), meta.Items[0].Size)
}

func TestMetaRemoveItemByName(t *testing.T) {
	meta := Meta{
		Items: []*ItemMeta{
			{Name: "main", NameExt: "mp4", Path: "main.mp4"},
			{Name: "small", NameExt: "mp4", Path: "small.mp4"},
		},
	}
	ok := meta.RemoveItemByName("main")
	assert.True(t, ok)
	assert.Equal(t, 1, len(meta.Items))
	assert.Equal(t, "small", meta.Items[0].Name)
}

func TestMetaExcessItems(t *testing.T) {
	keep := true
	w := &Workflow{
		Version: "2",
		Jobs: map[string]*WorkflowJob{
			"transcode": {
				Steps: []*WorkflowStep{
					{Uses: "video.transcode", With: map[string]any{"target": "main.mp4"}},
				},
			},
		},
		KeepOriginal: &keep,
	}
	meta := Meta{
		ManifestVersion: "2",
		Items: []*ItemMeta{
			{Name: "main", NameExt: "mp4", Path: "main.mp4"},
			{Name: "orphan", NameExt: "jpg", Path: "orphan.jpg"},
		},
	}
	excess := meta.ExcessItems(w)
	assert.Equal(t, 1, len(excess))
	assert.Equal(t, "orphan", excess[0].Name)
}

func TestMetaAttributes(t *testing.T) {
	meta := Meta{}
	meta.SetAttribute("dominant_color", "#ff0000")
	assert.Equal(t, "#ff0000", meta.GetAttribute("dominant_color"))
	assert.Nil(t, meta.GetAttribute("missing"))
}

func TestMetaIsConsistent(t *testing.T) {
	keep := true
	w := &Workflow{Version: "2", KeepOriginal: &keep}
	meta := Meta{ManifestVersion: "2"}
	assert.True(t, meta.IsConsistent(w))

	meta.ManifestVersion = "1"
	assert.False(t, meta.IsConsistent(w))
}

func TestMetaCleanSubItems(t *testing.T) {
	meta := Meta{
		Items: []*ItemMeta{
			{Name: "thumb", NameExt: "jpg"},
		},
		Attributes: map[string]any{"key": "val"},
	}
	meta.CleanSubItems()
	assert.Empty(t, meta.Items)
	assert.Nil(t, meta.Attributes)
}
