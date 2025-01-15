package v1

import "github.com/apfs-io/apfs/models"

// ManifestTaskStageFromModel creates new manifest task stage from model
func ManifestTaskStageFromModel(stage *models.ManifestTaskStage) (*ManifestTaskStage, error) {
	nStage := &ManifestTaskStage{
		Name:  stage.Name,
		Tasks: nil,
	}
	for _, ts := range stage.Tasks {
		task, err := ManifestTaskFromModel(ts)
		if err != nil {
			return nil, err
		}
		nStage.Tasks = append(nStage.Tasks, task)
	}
	return nStage, nil
}

// ToModel from protobuf object
func (m *ManifestTaskStage) ToModel() *models.ManifestTaskStage {
	return &models.ManifestTaskStage{
		Name:  m.Name,
		Tasks: m.modelTasks(),
	}
}

func (m *ManifestTaskStage) modelTasks() []*models.ManifestTask {
	list := make([]*models.ManifestTask, 0, len(m.Tasks))
	for _, ts := range m.Tasks {
		list = append(list, ts.ToModel())
	}
	return list
}
