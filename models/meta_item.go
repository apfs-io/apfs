package models

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
)

// ItemMeta holds metadata for a single file (original or derived artifact)
// within an object scope.
//
//easyjson:json
type ItemMeta struct {
	// Name is the base file name without extension (e.g. "main", "1").
	Name string `json:"name"`
	// NameExt is the file extension without a leading dot (e.g. "mp4", "jpg").
	NameExt string `json:"name_ext,omitempty"`

	// Path is the relative file path inside the object directory.
	// For flat artifacts this equals Fullname(); for nested ones it includes
	// sub-directories: "thumbs/1.jpg".
	Path string `json:"path,omitempty"`

	// Role is the job ID that produced this artifact, or "original" for the
	// source file. Useful for clients to understand provenance.
	Role string `json:"role,omitempty"`

	// Standard media properties — populated by probe/transcode steps.
	ContentType string `json:"content_type,omitempty"`
	HashID      string `json:"hashid,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Size        int64  `json:"size,omitempty"`
	Duration    int    `json:"duration,omitempty"` // seconds
	Bitrate     string `json:"bitrate,omitempty"`
	Codec       string `json:"codec,omitempty"`

	// Attributes holds additional domain-specific metadata for this file
	// (e.g. {"crc32": "a1b2c3d4", "dominant_color": "#ff0000"}).
	Attributes map[string]any `json:"attributes,omitempty"`

	// Type is the semantic object type (image, video, audio, etc.).
	Type ObjectType `json:"type,omitempty"`

	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// IsEmpty reports whether the item has no meaningful data.
func (m *ItemMeta) IsEmpty() bool {
	return m == nil || m.Name == ""
}

// Fullname returns the filename with extension (e.g. "main.mp4").
func (m *ItemMeta) Fullname() string {
	if m == nil {
		return ""
	}
	if strings.HasSuffix(m.Name, "."+m.NameExt) {
		return m.Name
	}
	if m.NameExt == "" {
		return m.Name
	}
	return m.Name + "." + m.NameExt
}

// EffectivePath returns Path if set, otherwise falls back to Fullname().
func (m *ItemMeta) EffectivePath() string {
	if m == nil {
		return ""
	}
	if m.Path != "" {
		return m.Path
	}
	return m.Fullname()
}

// ObjectTypeExt returns the file extension (without dot).
func (m *ItemMeta) ObjectTypeExt() string {
	if m.NameExt != "" {
		return m.NameExt
	}
	return strings.TrimPrefix(filepath.Ext(m.Name), ".")
}

// UpdateName sets Name and NameExt from a filename or path.
func (m *ItemMeta) UpdateName(name string) {
	if name == "" {
		return
	}
	base := filepath.Base(name)
	ext := strings.TrimPrefix(filepath.Ext(base), ".")
	m.Name = strings.TrimSuffix(base, filepath.Ext(base))
	m.NameExt = ext
	if m.Path == "" {
		m.Path = name
	}
}

// SetAttribute sets a key on the item's Attributes map.
func (m *ItemMeta) SetAttribute(key string, value any) {
	if m.Attributes == nil {
		m.Attributes = map[string]any{}
	}
	m.Attributes[key] = value
}

// GetAttribute returns an attribute value or nil.
func (m *ItemMeta) GetAttribute(key string) any {
	if m == nil || m.Attributes == nil {
		return nil
	}
	return m.Attributes[key]
}

// SetExt supports nested dot-notation paths for backward compatibility
// (e.g. "video.codec" creates {"video": {"codec": value}}).
// For simple keys prefer SetAttribute.
func (m *ItemMeta) SetExt(name string, value any) {
	if m.Attributes == nil {
		m.Attributes = map[string]any{}
	}
	keys := splitDotPath(name)
	if len(keys) == 1 {
		m.Attributes[name] = value
		return
	}
	ext := m.Attributes
	for i, key := range keys {
		if i == len(keys)-1 {
			ext[key] = value
			break
		}
		if v, ok := ext[key]; !ok || v == nil {
			next := map[string]any{}
			ext[key] = next
			ext = next
		} else {
			switch vmap := v.(type) {
			case map[string]any:
				ext = vmap
			default:
				next := map[string]any{}
				ext[key] = next
				ext = next
			}
		}
	}
}

// GetExt supports nested dot-notation paths for backward compatibility.
// For simple keys prefer GetAttribute.
func (m *ItemMeta) GetExt(name string) any {
	if m == nil || m.Attributes == nil {
		return nil
	}
	keys := splitDotPath(name)
	if len(keys) == 1 {
		if value, ok := m.Attributes[name]; ok {
			return value
		}
		return nil
	}
	ext := m.Attributes
	for i, key := range keys {
		v, ok := ext[key]
		if !ok {
			return nil
		}
		if i == len(keys)-1 {
			return v
		}
		switch vmap := v.(type) {
		case map[string]any:
			ext = vmap
		default:
			return nil
		}
	}
	return nil
}

func splitDotPath(s string) []string {
	// Inline split to avoid importing strings in hot path
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

// ExtJSON serialises Attributes to JSON. Returns "" when empty.
func (m *ItemMeta) ExtJSON() string {
	if m == nil || len(m.Attributes) == 0 {
		return ""
	}
	data, _ := json.Marshal(m.Attributes)
	if len(data) < 3 {
		return ""
	}
	return string(data)
}

// FromExtJSON deserialises JSON into Attributes.
func (m *ItemMeta) FromExtJSON(data []byte) error {
	return json.Unmarshal(data, &m.Attributes)
}
