package actionprocessors

import (
	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

type ActionProcessorSave struct{}

func (ActionProcessorSave) Name() string { return ActionSave }

func (ActionProcessorSave) Process(in converters.Input, out converters.Output, action *models.Action, imgReader ImageReader) error {
	// Uset as empty action to be able to save original image into target format
	if !action.ValueBool(ActionParamSave, false) {
		return imgReader.Close()
	}
	return nil
}
