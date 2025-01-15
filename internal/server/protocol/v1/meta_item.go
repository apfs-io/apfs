package v1

import "github.com/apfs-io/apfs/models"

// MetaItemFromModel creates new MetaItem from model
func MetaItemFromModel(metaItem *models.ItemMeta) *ItemMeta {
	return &ItemMeta{
		Name:    metaItem.Name,
		NameExt: metaItem.NameExt,

		Type:        metaItem.Type.String(),
		ContentType: metaItem.ContentType,
		HashId:      metaItem.HashID,
		Width:       int32(metaItem.Width),
		Height:      int32(metaItem.Height),
		Size:        uint64(metaItem.Size),
		Duration:    int64(metaItem.Duration),
		Bitrate:     metaItem.Bitrate,
		Codec:       metaItem.Codec,
		ExtJson:     metaItem.ExtJSON(),

		UpdatedAt: metaItem.UpdatedAt.UnixNano(),
	}
}

// ToModel from protobuf object
func (m *ItemMeta) ToModel() *models.ItemMeta {
	meta := &models.ItemMeta{
		Name:        m.Name,
		NameExt:     m.NameExt,
		Type:        models.ObjectType(m.Type),
		ContentType: m.ContentType,
		HashID:      m.HashId,
		Width:       int(m.Width),
		Height:      int(m.Height),
		Size:        int64(m.Size),
		Duration:    int(m.Duration),
		Bitrate:     m.Bitrate,
		Codec:       m.Codec,
	}
	meta.FromExtJSON([]byte(m.GetExtJson()))
	return meta
}
