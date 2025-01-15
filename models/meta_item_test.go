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
	}, item.Ext)
}
