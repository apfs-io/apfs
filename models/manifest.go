//
// @project apfs 2018 - 2019
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2019
//

package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Consts ...
const (
	MaxRetries = 10
)

//easyjson:json
type manifestObject struct {
	Version      string               `json:"version,omitempty"`
	ContentTypes []string             `json:"content_types,omitempty"`
	Stages       []*ManifestTaskStage `json:"stages,omitempty"`
}

// Manifest model object
type Manifest struct {
	Version      string               `json:"version,omitempty"`
	ContentTypes []string             `json:"content_types,omitempty"`
	Stages       []*ManifestTaskStage `json:"stages,omitempty"`
}

// IsEmpty manifest object
func (m *Manifest) IsEmpty() bool {
	if m == nil {
		return true
	}
	if m.Version != "" || len(m.ContentTypes) > 0 {
		return false
	}
	for _, stage := range m.Stages {
		if !stage.IsEmpty() {
			return false
		}
	}
	return true
}

// GetVersion returns the version of manifest safely
func (m *Manifest) GetVersion() string {
	if m == nil {
		return ""
	}
	return m.Version
}

// GetStages returns the list of stagest safely
func (m *Manifest) GetStages() []*ManifestTaskStage {
	if m == nil {
		return nil
	}
	return m.Stages
}

// TaskByTarget name of result
func (m *Manifest) TaskByTarget(name string) *ManifestTask {
	for _, stage := range m.GetStages() {
		if task := stage.TaskByTarget(name); task != nil {
			return task
		}
	}
	return nil
}

// TaskByID returns the task with such ID
func (m *Manifest) TaskByID(id string) *ManifestTask {
	for _, stage := range m.GetStages() {
		if task := stage.TaskByID(id); task != nil {
			return task
		}
	}
	return nil
}

// HasTaskByTarget name returns status
func (m *Manifest) HasTaskByTarget(name string) bool {
	return m != nil && m.TaskByTarget(name) != nil
}

// HasTask for target object in the manifest
func (m *Manifest) HasTask(id string) bool {
	for _, stage := range m.GetStages() {
		if stage.HasTask(id) {
			return true
		}
	}
	return false
}

// TaskCount returns total count of tasks
func (m *Manifest) TaskCount() int {
	totalCount := 0
	for _, stage := range m.GetStages() {
		totalCount += len(stage.Tasks)
	}
	return totalCount
}

// TargetCount returns total count of targets
func (m *Manifest) TargetCount() int {
	if m == nil {
		return 0
	}
	store := map[string]bool{}
	for _, stage := range m.Stages {
		for _, task := range stage.Tasks {
			if task.Target == "" {
				continue
			}
			if IsOriginal(task.Target) {
				store[OriginalFilename] = true
			} else {
				store[task.Target] = true
			}
		}
	}
	return len(store)
}

// IsValidContentType checks the content type
func (m *Manifest) IsValidContentType(contentType string) bool {
	fmt.Println("=== Content types: ", m.ContentTypes, "Content type: ", contentType)
	if m == nil || len(m.ContentTypes) == 0 {
		return true
	}
	for _, ct := range m.ContentTypes {
		if testContentType(contentType, ct) {
			return true
		}
	}
	return false
}

// UnmarshalJSON is custom method of json.Unmarshaler implementation
func (m *Manifest) UnmarshalJSON(data []byte) error {
	var manifestObj manifestObject
	if err := json.Unmarshal(data, &manifestObj); err != nil {
		return err
	}
	m.Version = manifestObj.Version
	m.ContentTypes = manifestObj.ContentTypes
	m.Stages = manifestObj.Stages
	m.PrepareInfo()
	return nil
}

// PrepareInfo of the manifest
func (m *Manifest) PrepareInfo() *Manifest {
	for _, stage := range m.GetStages() {
		stage.PrepareInfo()
	}
	return m
}

func testContentType(contentType, compareType string) bool {
	if compareType == "*" || compareType == "" || contentType == compareType {
		return true
	}
	if strings.HasSuffix(compareType, "/*") {
		return strings.HasPrefix(contentType, strings.TrimSuffix(compareType, "*"))
	}
	return false
}

var _ json.Unmarshaler = (*Manifest)(nil)
