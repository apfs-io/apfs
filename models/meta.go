package models

import (
	"slices"
	"time"
)

// Meta describes the artifacts that belong to one object in the storage.
// It is persisted as meta.json inside the object directory and cached in
// MetaCache.
//
// Meta intentionally does NOT track processing progress. Use ProcessingState
// for that. The separation keeps meta.json stable and cacheable while
// state.json can be updated on every processing step without invalidating
// the artifact cache.
//
//easyjson:json
type Meta struct {
	// Main holds metadata for the original/primary file.
	Main ItemMeta `json:"main"`

	// Items holds metadata for every derived artifact (transcodes, thumbnails,
	// previews, etc.).
	Items []*ItemMeta `json:"items,omitempty"`

	// Attributes is a free-form map for domain-specific object metadata that
	// does not fit into the structured fields above (e.g. dominant_colors,
	// detected_language, custom business data).
	Attributes map[string]any `json:"attributes,omitempty"`

	// Tags are user-defined labels copied from the Workflow at upload time.
	Tags []string `json:"tags,omitempty"`

	// Params stores arbitrary key→values supplied by the caller at upload time.
	Params map[string][]string `json:"params,omitempty"`

	// ManifestVersion is the Workflow.Version that was active when this meta
	// was last fully processed. Used to detect stale meta after manifest changes.
	ManifestVersion string `json:"manifest_version,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsEmpty reports whether the meta holds no meaningful data.
func (m *Meta) IsEmpty() bool {
	return m == nil || (m.Main.IsEmpty() && len(m.Items) == 0 && len(m.Tags) == 0)
}

// ItemByName returns the ItemMeta for the given name or path.
// Special names "@", "", "prim" refer to the main file.
func (m *Meta) ItemByName(name string) *ItemMeta {
	if IsOriginal(name) {
		return &m.Main
	}
	sourceName := SourceFilename(name, m.Main.ObjectTypeExt())
	for i, item := range m.Items {
		if item.Name == name || item.Name == sourceName || item.Path == name {
			return m.Items[i]
		}
	}
	return nil
}

// SetItem upserts item into Items by name. If an item with the same name
// already exists it is overwritten.
func (m *Meta) SetItem(item *ItemMeta) {
	if old := m.ItemByName(item.Name); old != nil {
		*old = *item
	} else {
		m.Items = append(m.Items, item)
	}
}

// RemoveItemByName removes the item with the given name or path from Items.
// Returns true if an item was found and removed.
func (m *Meta) RemoveItemByName(name string) bool {
	if m == nil {
		return false
	}
	sourceName := SourceFilename(name, m.Main.ObjectTypeExt())
	for i, it := range m.Items {
		if it.Name == name || it.Name == sourceName || it.Path == name {
			m.Items = slices.Delete(m.Items, i, i+1)
			return true
		}
	}
	return false
}

// ExcessItems returns items whose name or path is not referenced by any job
// target in the workflow. Used to detect leftover artifacts after a manifest change.
func (m *Meta) ExcessItems(w *Workflow) []*ItemMeta {
	if m == nil || w == nil {
		return nil
	}
	excess := make([]*ItemMeta, 0)
	for _, item := range m.Items {
		found := false
		for _, job := range w.Jobs {
			for _, step := range job.Steps {
				target, _ := step.With["target"].(string)
				if target != "" && (target == item.Name || target == item.Path || target == item.Fullname()) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			excess = append(excess, item)
		}
	}
	return excess
}

// MissingJobTargets returns job IDs whose declared step targets are absent from Items.
func (m *Meta) MissingJobTargets(w *Workflow) []string {
	if m == nil || w == nil {
		return nil
	}
	missing := make([]string, 0)
	for jobID, job := range w.Jobs {
		if job == nil {
			continue
		}
		for _, step := range job.Steps {
			target, _ := step.With["target"].(string)
			if target == "" {
				continue
			}
			if m.ItemByName(target) == nil {
				missing = append(missing, jobID)
				break
			}
		}
	}
	slices.Sort(missing)
	return missing
}

// IsConsistent reports whether this meta is up-to-date with the given workflow.
// A meta is consistent when:
//   - ManifestVersion matches the workflow version, AND
//   - there are no excess or missing derived items for the workflow jobs.
func (m *Meta) IsConsistent(w *Workflow) bool {
	if m == nil {
		return false
	}
	if w == nil {
		return true // no workflow → nothing to check
	}
	if m.ManifestVersion != w.Version {
		return false
	}
	return len(m.ExcessItems(w)) == 0 && len(m.MissingJobTargets(w)) == 0
}

// SetAttribute sets a free-form attribute on the object.
func (m *Meta) SetAttribute(key string, value any) {
	if m.Attributes == nil {
		m.Attributes = map[string]any{}
	}
	m.Attributes[key] = value
}

// GetAttribute returns a free-form attribute value or nil.
func (m *Meta) GetAttribute(key string) any {
	if m == nil || m.Attributes == nil {
		return nil
	}
	return m.Attributes[key]
}

// CleanSubItems resets all derived artifacts and free-form attributes,
// leaving only the main file metadata intact. Used before re-processing.
func (m *Meta) CleanSubItems() {
	if m == nil {
		return
	}
	m.Items = m.Items[:0]
	m.Attributes = nil
	m.Main.Attributes = nil
}
