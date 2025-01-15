package actionprocessors

import (
	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

var errInvalidImageSize = errors.New(`invalid image size`)

type ActionProcessorSizeValidator struct{}

func (ActionProcessorSizeValidator) Name() string { return ActionValidateSize }

func (ActionProcessorSizeValidator) Process(_ converters.Input, _ converters.Output, action *models.Action, imgReader ImageReader) (err error) {
	rect := imgReader.Image().Bounds()
	w, h, wm, hm := action.ValueInt32(ActionParamWidth, 0), action.ValueInt32(ActionParamHeight, 0),
		action.ValueInt32(ActionParamMaxWidth, 0), action.ValueInt32(ActionParamMaxHeight, 0)
	if !isSuitsSize(w, h, wm, hm, int32(rect.Dx()), int32(rect.Dy())) {
		err = errors.Wrapf(errInvalidImageSize, `%dx%d`, rect.Dx(), rect.Dy())
	}
	return err
}

// IsSuitsSize checks if no need to resize
func isSuitsSize(width, height, maxWidth, maxHeight, currentWidth, currentHeight int32) bool {
	if maxWidth > width {
		if maxWidth < currentWidth || width > currentWidth {
			return false
		}
	} else if width > 0 && width != currentWidth {
		return false
	}
	if maxHeight > height {
		if maxHeight < currentHeight || height > currentHeight {
			return false
		}
	} else if height > 0 && height != currentHeight {
		return false
	}
	return true
}
