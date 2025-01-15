package actionprocessors

import (
	"github.com/disintegration/imaging"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

type ActionProcessorContrast struct{}

func (ActionProcessorContrast) Name() string { return ActionContrast }

func (ActionProcessorContrast) Process(in converters.Input, out converters.Output, action *models.Action, imgReader ImageReader) error {
	if v := action.ValueFloat64(ActionParamValue, 0); v != 0 {
		img := imaging.AdjustContrast(imgReader.Image(), v)
		imgReader.SetImage(img)
	}
	return nil
}
