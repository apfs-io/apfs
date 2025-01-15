package actionprocessors

import (
	"github.com/disintegration/imaging"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

type ActionProcessorBrightness struct{}

func (ActionProcessorBrightness) Name() string { return ActionBrightness }

func (ActionProcessorBrightness) Process(in converters.Input, out converters.Output, action *models.Action, imgReader ImageReader) error {
	if v := action.ValueFloat64(ActionParamValue, 0); v != 0 {
		img := imaging.AdjustBrightness(imgReader.Image(), v)
		imgReader.SetImage(img)
	}
	return nil
}
