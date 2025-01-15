package v1

import (
	"github.com/apfs-io/apfs/models"
)

// ManifestFromModel creates new manifest from model
func ManifestFromModel(man *models.Manifest) (*Manifest, error) {
	manifest := &Manifest{
		Version:      man.Version,
		ContentTypes: man.ContentTypes,
	}
	for _, stage := range man.Stages {
		mStage, err := ManifestTaskStageFromModel(stage)
		if err != nil {
			return nil, err
		}
		manifest.Stages = append(manifest.Stages, mStage)
	}
	return manifest, nil
}

// ToModel from protobuf object
func (m *Manifest) ToModel() *models.Manifest {
	return &models.Manifest{
		Version:      m.Version,
		ContentTypes: m.ContentTypes,
		Stages:       m.modelStages(),
	}
}

func (m *Manifest) modelStages() []*models.ManifestTaskStage {
	list := make([]*models.ManifestTaskStage, 0, len(m.Stages))
	for _, st := range m.Stages {
		list = append(list, st.ToModel())
	}
	return list
}
