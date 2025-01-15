//
// @project apfs 2018 - 2020
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2020
//

package models

import (
	"errors"
	"slices"
	"time"

	"go.uber.org/zap"
)

// ErrInvalidMetaMainItem error
var ErrInvalidMetaMainItem = errors.New(`invalid meta main item`)

// MetaTaskInfo contains information of one action item
//
//easyjson:json
type MetaTaskInfo struct {
	ID             string       `json:"id"`
	Attempts       int          `json:"attempts,omitempty"`
	Status         ObjectStatus `json:"status,omitempty"`
	StatusMessage  string       `json:"status_message,omitempty"`
	TargetItemName string       `json:"target_item_name,omitempty"`
	UpdatedAt      time.Time    `json:"updated_at,omitempty"`
}

// Meta information of the file object
//
//easyjson:json
type Meta struct {
	Main            ItemMeta            `json:"main"`
	Tasks           []*MetaTaskInfo     `json:"tasks,omitempty"` // List of complete actions
	Items           []*ItemMeta         `json:"items,omitempty"`
	Tags            []string            `json:"tags,omitempty"`
	Params          map[string][]string `json:"params,omitempty"`
	ManifestVersion string              `json:"manifest_version,omitempty"`
	CreatedAt       time.Time           `json:"created_at,omitempty"`
	UpdatedAt       time.Time           `json:"updated_at,omitempty"`
}

// IsEmpty manifest object
func (m *Meta) IsEmpty() bool {
	if m == nil {
		return true
	}
	return m.Main.IsEmpty() && len(m.Items) < 1 && len(m.Tags) < 1
}

// ItemByName returns subfile information
func (m *Meta) ItemByName(name string) *ItemMeta {
	if IsOriginal(name) {
		return &m.Main
	}
	sourceName := SourceFilename(name, m.Main.ObjectTypeExt())
	for i, item := range m.Items {
		if item.Name == name || item.Name == sourceName {
			return m.Items[i]
		}
	}
	return nil
}

// IsComplete manifest processed
func (m *Meta) IsComplete(manifest *Manifest, stages ...string) bool {
	completed := 0
	for _, stage := range manifest.Stages {
		slices.Contains(stages, stage.Name)
		if len(stages) > 0 && !slices.Contains(stages, stage.Name) {
			continue
		}
		for _, task := range stage.Tasks {
			if !m.IsCompleteTask(task) {
				return false
			}
		}
		completed++
	}
	return (len(stages) > 0 && completed == len(stages)) ||
		(len(stages) == 0 && completed == len(manifest.Stages))
}

// IsProcessingComplete with any status
func (m *Meta) IsProcessingComplete(manifest *Manifest, stages ...string) bool {
	completed := 0
	if manifest == nil {
		zap.L().Error("IsProcessingComplete invalid manifest")
		return false // TODO: logick check, if no manifest maybe all done?
	}
	for _, stage := range manifest.Stages {
		if len(stages) > 0 && !slices.Contains(stages, stage.Name) {
			continue
		}
		for _, task := range stage.Tasks {
			if !m.IsProcessingCompleteTask(task) {
				return false
			}
		}
		completed++
	}
	return (len(stages) > 0 && completed == len(stages)) ||
		(len(stages) == 0 && completed == len(manifest.Stages))
}

// IsCompleteTask returns complition status of the task
func (m *Meta) IsCompleteTask(task *ManifestTask) bool {
	for _, complete := range m.Tasks {
		if complete.ID == task.ID && complete.Status.IsProcessed() {
			return true
		}
	}
	return false
}

// IsProcessingCompleteTask returns finished complition status of the task
func (m *Meta) IsProcessingCompleteTask(task *ManifestTask, maxRetries ...int) bool {
	for _, complete := range m.Tasks {
		if complete.ID == task.ID {
			if complete.Status.IsError() || (complete.Status.IsProcessing() && time.Since(complete.UpdatedAt) > time.Minute*5) {
				return len(maxRetries) == 0 || complete.Attempts > maxRetries[0]
			}
			return true
		}
	}
	return false
}

// Complete marks action as complete
func (m *Meta) Complete(itemMeta *ItemMeta, task *ManifestTask, err error) {
	attempts := 0
	// Remove task if was stored
	for i, prevTask := range m.Tasks {
		if prevTask.ID == task.ID {
			attempts = prevTask.Attempts + 1
			m.Tasks = append(m.Tasks[:i], m.Tasks[i+1:]...)
			break
		}
	}
	taskInfo := &MetaTaskInfo{
		ID:             task.ID,
		Status:         StatusOK,
		Attempts:       attempts,
		TargetItemName: itemMeta.Fullname(),
		UpdatedAt:      time.Now(),
	}
	if err != nil {
		taskInfo.Status = StatusError
		taskInfo.StatusMessage = err.Error()
	}
	m.Tasks = append(m.Tasks, taskInfo)

	if task.ID != "" {
		if taskInfo.Status.IsProcessed() {
			itemMeta.MarkAsComplete(task.ID)
		} else {
			itemMeta.MarkAsIncomplete(task.ID)
		}
	}
}

// ResetCompletion state
func (m *Meta) ResetCompletion() {
	if m != nil && len(m.Tasks) > 0 {
		m.Tasks = m.Tasks[:0]
	}
}

// ErrorTaskCount returns count of tasks which returns the error
func (m *Meta) ErrorTaskCount() (errorCount int) {
	if m == nil {
		return 0
	}
	for _, task := range m.Tasks {
		if task.Status.IsError() {
			errorCount++
		}
	}
	return errorCount
}

// ExcessItems returns the list of items which is not present in the Manifest
func (m *Meta) ExcessItems(manifest *Manifest) []*ItemMeta {
	if m == nil || manifest == nil {
		return nil
	}
	items := make([]*ItemMeta, 0, len(m.Items))
	for _, item := range m.Items {
		if !manifest.HasTaskByTarget(item.Fullname()) {
			items = append(items, item)
		}
	}
	return items
}

// RemoveExcessTasks from the meta object
func (m *Meta) RemoveExcessTasks(manifest *Manifest) {
	tasks := make([]*MetaTaskInfo, 0, len(m.Tasks))
	for _, task := range m.Tasks {
		manifestTask := manifest.TaskByID(task.ID)
		if manifestTask == nil {
			continue
		}
		if task.TargetItemName == "" || task.TargetItemName == "." || task.TargetItemName == "@" {
			task.TargetItemName = manifestTask.Target
		}
		if task.Status.IsError() || task.Status.IsProcessing() ||
			task.TargetItemName == "" || m.ItemByName(task.TargetItemName) != nil {
			tasks = append(tasks, task)
		}
	}
	m.Tasks = tasks
}

// IsConsistent meta-information and manifest state
func (m *Meta) IsConsistent(manifest *Manifest) bool {
	return m != nil && (manifest == nil ||
		m.ManifestVersion == manifest.Version &&
			m.IsProcessingComplete(manifest) &&
			len(m.ExcessItems(manifest)) == 0 &&
			// If the amount of target items corresponds to processed items.
			// Tasks with error don't save the item so we can skip them.
			(manifest.TargetCount()-m.ErrorTaskCount()) <= len(m.Items))
}

// RemoveItemByName meta information about item
func (m *Meta) RemoveItemByName(name string) (bool, error) {
	if m == nil {
		return false, nil
	}
	sourceName := SourceFilename(name, m.Main.ObjectTypeExt())
	for i, it := range m.Items {
		if it.Name == name || it.Name == sourceName {
			if i == len(m.Items)-1 {
				m.Items = m.Items[:i]
			} else {
				m.Items = append(m.Items[:i], m.Items[i+1:]...)
			}
			for _, taskID := range it.TaskID {
				_ = m.RemoveTaskByID(taskID)
			}
			return true, nil
		}
	}
	return false, nil
}

// RemoveTaskByID meta information about task
func (m *Meta) RemoveTaskByID(id string) bool {
	for i, it := range m.Tasks {
		if it.ID == id {
			if i == len(m.Tasks)-1 {
				m.Tasks = m.Tasks[:i]
			} else {
				m.Tasks = append(m.Tasks[:i], m.Tasks[i+1:]...)
			}
			return true
		}
	}
	return false
}

// SetItem meta information
func (m *Meta) SetItem(item *ItemMeta) {
	if old := m.ItemByName(item.Name); old != nil {
		*old = *item
	} else {
		m.Items = append(m.Items, item)
	}
}

// CleanSubItems information from the object
func (m *Meta) CleanSubItems() {
	if m == nil {
		return
	}
	m.Tasks = m.Tasks[:0]
	m.Items = m.Items[:0]
	m.Main.TaskID = m.Main.TaskID[:0]
	m.Main.Ext = map[string]any{}
}
