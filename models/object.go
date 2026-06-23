//
// @project apfs 2018 - 2019
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2019
//

package models

import (
	"path/filepath"
	"time"

	"github.com/geniusrabbit/gosql/v2"
)

// Object structure which describes the paticular file attached to advertisement
// Image advertisement: Title=Image title, Description=My description
//
//	      ID,  Bucket,         path,             HashID,  size,  type, content_type,                          meta
//	File:  1, 'image', 'images/a/c', dhg321h3ndp43u2hfc, 64322, image,   image/jpeg, {"width": 300, "height": 250}
//	File:  2, 'image', 'images/a/c', xxg321h3xxx43u2hfc, 44322, video,  video/x-mp4, {"width": 300, "height": 250, "duration": 11132ms}
//
//easyjson:json
type Object struct {
	ID     string `json:"id" gorm:"primary_key"` // Unical ID in storage
	Bucket string `json:"bucket"`
	Path   string `json:"path"`
	HashID string `json:"hashid" gorm:"index:hashid;column:hashid"`

	Status        ObjectStatus `json:"status"`
	StatusMessage string       `json:"status_message,omitempty"`

	ContentType string                          `json:"content_type"`
	Type        ObjectType                      `json:"type"`
	Tags        gosql.NullableJSONArray[string] `json:"tags,omitempty"`
	Meta        gosql.NullableJSON[Meta]        `json:"meta,omitempty"`
	Workflow    gosql.NullableJSON[Workflow]    `json:"workflow,omitempty"`
	Size        uint64                          `json:"size"` // Size in bytes

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ObjectID returns the Identificator of the object
func (o *Object) ObjectID() string {
	if o == nil {
		return ""
	}
	return o.ID
}

// TableName of the object in the database
func (o *Object) TableName() string {
	return "object"
}

// IncompleteJobs returns the list of workflow jobs whose output targets
// are not yet present in the object's meta items.
func (o *Object) IncompleteJobs() (resp []string) {
	meta := o.Meta.Data
	wf := o.Workflow.Data
	for jobID, job := range wf.Jobs {
		if job == nil {
			continue
		}
		for _, step := range job.Steps {
			target, _ := step.With["target"].(string)
			if target == "" {
				continue
			}
			if meta.ItemByName(target) == nil {
				resp = append(resp, jobID)
				break
			}
		}
	}
	return resp
}

// PathByName returns the full path to the item
func (o *Object) PathByName(name string) string {
	return filepath.Join(o.Path, o.Meta.Data.ItemByName(name).Fullname())
}
