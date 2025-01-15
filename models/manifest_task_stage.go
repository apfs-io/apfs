package models

import (
	"path/filepath"
	"strings"

	"github.com/demdxx/gocast/v2"
)

// ManifestTaskStage file processing
//
//easyjson:json
type ManifestTaskStage struct {
	Name  string          `json:"name"`
	Tasks []*ManifestTask `json:"tasks,omitempty"`
}

// IsEmpty manifest object
func (stage *ManifestTaskStage) IsEmpty() bool {
	return stage == nil || len(stage.Tasks) == 0
}

// TaskByTarget name of result
func (stage *ManifestTaskStage) TaskByTarget(name string) *ManifestTask {
	alternativeName := strings.TrimRight(name, filepath.Ext(name))
	for _, task := range stage.Tasks {
		if task.Target == name || task.Target == alternativeName {
			return task
		}
	}
	return nil
}

// TaskByID returns the task with ID
func (stage *ManifestTaskStage) TaskByID(id string) *ManifestTask {
	for _, task := range stage.Tasks {
		if task.ID == id {
			return task
		}
	}
	return nil
}

// HasTask for target object in the manifest
func (stage *ManifestTaskStage) HasTask(id string) bool {
	return stage.TaskByID(id) != nil
}

// PrepareInfo of the manifest stage
func (stage *ManifestTaskStage) PrepareInfo() {
	present := map[string]int{}
	for i, task := range stage.Tasks {
		if task.ID != `` {
			continue
		}
		objectPath := task.Target
		if objectPath == "" {
			objectPath = task.Source
		}
		ext := filepath.Ext(objectPath)
		prefix := SourceFilename(strings.TrimSuffix(objectPath, ext), ext)
		if stage.Name != `` {
			prefix = stage.Name + `:` + prefix
		}
		id := present[prefix]
		task.ID = prefix + ":" + gocast.Str(id+1)
		present[prefix] = id + 1
		stage.Tasks[i] = task
	}
}
