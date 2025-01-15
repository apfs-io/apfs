package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveExcessTasks(t *testing.T) {
	manifest := &Manifest{
		Version: "test",
		Stages: []*ManifestTaskStage{
			{
				Name: "stage1",
				Tasks: []*ManifestTask{
					{ID: "i1", Target: "k1.png"},
					{ID: "i2", Target: "k2.png"},
					{ID: "i3", Target: "k3.png"},
				},
			},
		},
	}
	meta := Meta{
		Main: ItemMeta{Name: "main"},
		Tasks: []*MetaTaskInfo{
			{ID: "i1", Status: StatusOK},
			{ID: "i2", Status: StatusOK},
			{ID: "i3", Status: StatusOK},
		},
		Items: []*ItemMeta{
			{Name: "k2.png", NameExt: "png", TaskID: []string{"i2"}},
		},
	}

	assert.Equal(t, 0, len(meta.ExcessItems(manifest)))

	meta.RemoveExcessTasks(manifest)
	assert.Equal(t, 1, len(meta.Tasks))
	assert.ElementsMatch(t, []*MetaTaskInfo{{
		ID: "i2", Status: StatusOK, TargetItemName: "k2.png"}}, meta.Tasks)
}
