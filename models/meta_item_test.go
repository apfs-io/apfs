package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItemMetaSetGetExt(t *testing.T) {
	var item ItemMeta

	item.SetExt("test.value.item", true)
	item.SetExt("test.value.test", nil)

	assert.Equal(t, true, item.GetExt("test.value.item"))
	assert.Nil(t, item.GetExt("test.value.nil"))
	assert.Nil(t, item.GetExt("test.valuenil.value"))
	assert.Nil(t, item.GetExt("test.value.item.undefined"))
	assert.Nil(t, item.GetExt("nil"))
	assert.Equal(t, map[string]any{
		"test": map[string]any{
			"value": map[string]any{
				"item": true,
				"test": nil,
			},
		},
	}, item.Attributes)
}

func TestItemMetaFullname(t *testing.T) {
	item := ItemMeta{Name: "main", NameExt: "mp4"}
	assert.Equal(t, "main.mp4", item.Fullname())

	item2 := ItemMeta{Name: "main.mp4", NameExt: "mp4"}
	assert.Equal(t, "main.mp4", item2.Fullname())

	item3 := ItemMeta{Name: "thumb", NameExt: ""}
	assert.Equal(t, "thumb", item3.Fullname())
}

func TestItemMetaUpdateName(t *testing.T) {
	var item ItemMeta
	item.UpdateName("thumbs/1.jpg")
	assert.Equal(t, "1", item.Name)
	assert.Equal(t, "jpg", item.NameExt)
	assert.Equal(t, "thumbs/1.jpg", item.Path)
}

func TestItemMetaEffectivePath(t *testing.T) {
	item := ItemMeta{Name: "main", NameExt: "mp4"}
	assert.Equal(t, "main.mp4", item.EffectivePath())

	item2 := ItemMeta{Name: "1", NameExt: "jpg", Path: "thumbs/1.jpg"}
	assert.Equal(t, "thumbs/1.jpg", item2.EffectivePath())
}
