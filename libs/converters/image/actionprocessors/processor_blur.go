package actionprocessors

import (
	"github.com/disintegration/imaging"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

type ActionProcessorBlur struct{}

func (ActionProcessorBlur) Name() string { return ActionBlur }

func (ActionProcessorBlur) Process(in converters.Input, out converters.Output, action *models.Action, imgReader ImageReader) error {
	if v := action.ValueFloat64(ActionParamValue, 0); v > 0 {
		img := imaging.Blur(imgReader.Image(), v)
		imgReader.SetImage(img)
	}
	return nil
}
