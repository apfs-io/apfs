package client

import (
	"encoding/json"
	"time"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/models"
)

// ItemMeta describes a single artifact file produced by the processing pipeline.
// It is a client-friendly projection of the internal models.ItemMeta type.
type ItemMeta struct {
	Name        string
	Path        string
	Role        string
	ContentType string
	Type        models.ObjectType
	Width       int
	Height      int
	Size        int64
	Duration    int64
	Bitrate     string
	Codec       string
	Attributes  map[string]any
	UpdatedAt   time.Time
}

// Meta holds the full metadata for an object and its produced artifacts.
type Meta struct {
	Main            ItemMeta
	Items           []*ItemMeta
	Tags            []string
	Attributes      map[string]any
	ManifestVersion string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// metaFromProto converts the generated proto Meta type to the client Meta type.
func metaFromProto(p *protocol.Meta) *Meta {
	if p == nil {
		return nil
	}
	m := &Meta{
		Main:            itemMetaFromProto(p.GetMain()),
		Tags:            append([]string{}, p.GetTags()...),
		ManifestVersion: p.GetManifestVersion(),
		CreatedAt:       time.Unix(0, p.GetCreatedAt()),
		UpdatedAt:       time.Unix(0, p.GetUpdatedAt()),
	}
	_ = json.Unmarshal([]byte(p.GetAttributesJson()), &m.Attributes)
	for _, item := range p.GetItems() {
		m.Items = append(m.Items, itemMetaFromProtoPtr(item))
	}
	return m
}

func itemMetaFromProto(p *protocol.ItemMeta) ItemMeta {
	if p == nil {
		return ItemMeta{}
	}
	item := ItemMeta{
		Name:        p.GetName(),
		Path:        p.GetPath(),
		Role:        p.GetRole(),
		ContentType: p.GetContentType(),
		Type:        models.ObjectType(p.GetType()),
		Width:       int(p.GetWidth()),
		Height:      int(p.GetHeight()),
		Size:        int64(p.GetSize()),
		Duration:    p.GetDuration(),
		Bitrate:     p.GetBitrate(),
		Codec:       p.GetCodec(),
		UpdatedAt:   time.Unix(0, p.GetUpdatedAt()),
	}
	src := p.GetAttributesJson()
	if src == "" {
		src = p.GetExtJson()
	}
	if src != "" {
		_ = json.Unmarshal([]byte(src), &item.Attributes)
	}
	return item
}

func itemMetaFromProtoPtr(p *protocol.ItemMeta) *ItemMeta {
	if p == nil {
		return nil
	}
	item := itemMetaFromProto(p)
	return &item
}
