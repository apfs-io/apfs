package v1

import "github.com/apfs-io/apfs/models"

// ManifestTaskFromModel creates new manifest task from model
func ManifestTaskFromModel(ts *models.ManifestTask) (*ManifestTask, error) {
	task := &ManifestTask{
		Id:      ts.ID,
		Source:  ts.Source,
		Target:  ts.Target,
		Type:    ts.Type.String(),
		Actions: nil,
	}
	for _, act := range ts.Actions {
		action, err := ActionFromModel(act)
		if err != nil {
			return nil, err
		}
		task.Actions = append(task.Actions, action)
	}
	return task, nil
}

// ToModel from protobuf object
func (m *ManifestTask) ToModel() *models.ManifestTask {
	return &models.ManifestTask{
		ID:      m.Id,
		Source:  m.Source,
		Target:  m.Target,
		Type:    models.ObjectType(m.Type),
		Actions: m.modelActions(),
	}
}

func (m *ManifestTask) modelActions() []*models.Action {
	list := make([]*models.Action, 0, len(m.Actions))
	for _, act := range m.Actions {
		list = append(list, act.ToModel())
	}
	return list
}
