package actionprocessors

import (
	"github.com/disintegration/imaging"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

type ActionProcessorFill struct{}

func (ActionProcessorFill) Name() string { return ActionFill }

func (ActionProcessorFill) Process(in converters.Input, out converters.Output, action *models.Action, imgReader ImageReader) error {
	rect := imgReader.Image().Bounds()
	w, h := int(action.ValueInt32(ActionParamWidth, 0)), int(action.ValueInt32(ActionParamHeight, 0))
	if action.MustExecute || w != rect.Dx() || h != rect.Dy() {
		anchor := action.ValueString(ActionParamAnchor, "")
		filter := action.ValueString(ActionParamFilter, "")
		img := imaging.Fill(imgReader.Image(), w, h,
			AnchorByString(anchor, imaging.Center),
			ResampleFilterByString(filter, imaging.Lanczos))
		imgReader.SetImage(img)
	}
	return nil
}
