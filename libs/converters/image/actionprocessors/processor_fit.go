package actionprocessors

import (
	"github.com/disintegration/imaging"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

type ActionProcessorFit struct{}

func (ActionProcessorFit) Name() string { return ActionFit }

func (ActionProcessorFit) Process(in converters.Input, out converters.Output, action *models.Action, imgReader ImageReader) error {
	rect := imgReader.Image().Bounds()
	w, h := int(action.ValueInt32(ActionParamWidth, 0)), int(action.ValueInt32(ActionParamHeight, 0))
	if action.MustExecute || w < rect.Dx() || h < rect.Dy() {
		filter := action.ValueString(ActionParamFilter, "")
		img := imaging.Fit(imgReader.Image(), w, h, ResampleFilterByString(filter, imaging.Lanczos))
		imgReader.SetImage(img)
	}
	return nil
}
