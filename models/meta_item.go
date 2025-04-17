package models

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
)

// ItemMeta information about one object element
//
//easyjson:json
type ItemMeta struct {
	TaskID []string `json:"task_id,omitempty"`

	Name    string `json:"name"`
	NameExt string `json:"name_ext,omitempty"`

	Type        ObjectType     `json:"type,omitempty"`
	ContentType string         `json:"content_type,omitempty"`
	HashID      string         `json:"hashid,omitempty"`
	Width       int            `json:"width,omitempty"`
	Height      int            `json:"height,omitempty"`
	Size        int64          `json:"size,omitempty"`
	Duration    int            `json:"duration,omitempty"`
	Bitrate     string         `json:"bitrate,omitempty"`
	Codec       string         `json:"codec,omitempty"`
	Ext         map[string]any `json:"ext,omitempty"`

	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// IsTaskProcessed checks if the task was applied to the Item
func (m *ItemMeta) IsTaskProcessed(id string) bool {
	for _, taskID := range m.TaskID {
		if taskID == id {
			return true
		}
	}
	return false
}

// MarkAsComplete task
func (m *ItemMeta) MarkAsComplete(id string) {
	if !m.IsTaskProcessed(id) {
		m.TaskID = append(m.TaskID, id)
	}
}

// MarkAsIncomplete task
func (m *ItemMeta) MarkAsIncomplete(id string) {
	for i := 0; i < len(m.TaskID); i++ {
		if m.TaskID[i] == id {
			m.TaskID = append(m.TaskID[:i], m.TaskID[i+1:]...)
			break
		}
	}
}

// IsEmpty manifest object
func (m *ItemMeta) IsEmpty() bool {
	return m == nil || m.Name == ""
}

// UpdateName information
func (m *ItemMeta) UpdateName(name string) {
	if name != "" {
		m.Name = filepath.Base(name)
		m.NameExt = strings.Trim(filepath.Ext(name), ".")
	}
}

// ExtJSON variable
func (m *ItemMeta) ExtJSON() string {
	if m.Ext == nil || len(m.Ext) < 1 {
		return ""
	}
	data, _ := json.Marshal(m.Ext)
	if len(data) < 4 {
		return ""
	}
	return string(data)
}

// FromExtJSON data
func (m *ItemMeta) FromExtJSON(data []byte) error {
	return json.Unmarshal(data, &m.Ext)
}

// Fullname of the file
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

// ObjectTypeExt returns file extension
func (m *ItemMeta) ObjectTypeExt() string {
	if m.NameExt == "" {
		m.NameExt = filepath.Ext(m.Name)
	}
	return m.NameExt
}

// SetExt context value
func (m *ItemMeta) SetExt(name string, value any) {
	if m.Ext == nil {
		m.Ext = map[string]any{}
	}
	keys := strings.Split(name, ".")
	if len(keys) == 1 {
		m.Ext[name] = value
	} else {
		ext := m.Ext
		for i, key := range keys {
			if i == len(keys)-1 {
				ext[key] = value
				break
			}
			if v, ok := ext[key]; !ok || v == nil {
				newMap := map[string]any{}
				ext[key] = newMap
				ext = newMap
			} else {
				switch vext := v.(type) {
				case map[string]any:
					ext = vext
				default:
					ext[key] = map[string]any{}
				}
			}
		}
	}
}

// GetExt value from context
func (m *ItemMeta) GetExt(name string) any {
	if m.Ext == nil {
		return nil
	}
	keys := strings.Split(name, ".")
	if len(keys) == 1 {
		if value, ok := m.Ext[name]; ok && value != nil {
			return value
		}
		return nil
	}

	ext := m.Ext
	for i, key := range keys {
		if i == len(keys)-1 {
			if value, ok := ext[key]; ok && value != nil {
				return value
			}
		} else if v, ok := ext[key]; ok && v != nil {
			switch vext := v.(type) {
			case map[string]any:
				ext = vext
			default:
				return nil
			}
		} else {
			break
		}
	}
	return nil
}
