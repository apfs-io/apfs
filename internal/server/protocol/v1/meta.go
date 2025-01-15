package v1

import (
	"time"

	"github.com/apfs-io/apfs/models"
)

// MetaFromModel creates new meta from model
func MetaFromModel(meta *models.Meta) *Meta {
	mt := &Meta{
		ManifestVersion: meta.ManifestVersion,
		Main:            MetaItemFromModel(&meta.Main),
		Tags:            meta.Tags,
		CreatedAt:       meta.CreatedAt.UnixNano(),
		UpdatedAt:       meta.UpdatedAt.UnixNano(),
	}
	for _, mtItem := range meta.Items {
		mt.Items = append(mt.Items, MetaItemFromModel(mtItem))
	}
	return mt
}

// ToModel from protobuf object
func (m *Meta) ToModel() *models.Meta {
	var items []*models.ItemMeta
	for _, it := range m.GetItems() {
		if mit := it.ToModel(); mit != nil {
			items = append(items, mit)
		}
	}
	return &models.Meta{
		ManifestVersion: m.ManifestVersion,
		Main:            metaItemValue(m.GetMain().ToModel()),
		Items:           items,
		Tags:            m.GetTags(),
		CreatedAt:       time.Unix(0, m.CreatedAt),
		UpdatedAt:       time.Unix(0, m.UpdatedAt),
	}
}

func metaItemValue(item *models.ItemMeta) models.ItemMeta {
	if item == nil {
		return models.ItemMeta{}
	}
	return *item
}
