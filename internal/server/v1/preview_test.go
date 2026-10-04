package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apfs-io/apfs/models"
)

func TestPreviewImageItem(t *testing.T) {
	jpeg := &models.Meta{
		Items: []*models.ItemMeta{
			{Name: "preview", NameExt: "jpg", ContentType: "image/jpeg"},
		},
	}
	item := previewImageItem(jpeg)
	assert.NotNil(t, item)
	assert.Equal(t, "image/jpeg", previewContentType(item))

	video := &models.Meta{
		Items: []*models.ItemMeta{
			{Name: "preview", NameExt: "mp4", ContentType: "video/mp4", Type: models.TypeVideo},
		},
	}
	assert.Nil(t, previewImageItem(video))

	byExt := &models.Meta{
		Items: []*models.ItemMeta{
			{Name: "preview", NameExt: "jpg"},
		},
	}
	item = previewImageItem(byExt)
	assert.NotNil(t, item)
	assert.Equal(t, "image/jpeg", previewContentType(item))

	assert.Nil(t, previewImageItem(&models.Meta{}))
	assert.Nil(t, previewImageItem(nil))
}
