//
// @project apfs 2018 - 2020
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2020
//

package image

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/libs/converters/image/actionprocessors"
	"github.com/apfs-io/apfs/libs/converters/image/imagereader"
	"github.com/apfs-io/apfs/models"
)

var (
	errImageDecode = errors.New(`invalid image decode`)
)

// ImageReader basic image reading desctiption
type ImageReader = actionprocessors.ImageReader

// ImageProcessor basic processor interface
type ImageProcessor interface {
	Name() string
	Process(in converters.Input, out converters.Output, action *models.Action, imgReader ImageReader) error
}

// Converter contains whole functions to convert image
type Converter struct {
	processors map[string]ImageProcessor
}

// NewDefaultConverter with all default actions
func NewDefaultConverter() *Converter {
	return (&Converter{}).RegisterImageProcessor(
		&actionprocessors.ActionProcessorSizeValidator{},
		&actionprocessors.ActionProcessorResize{},
		&actionprocessors.ActionProcessorFit{},
		&actionprocessors.ActionProcessorFill{},
		&actionprocessors.ActionProcessorSharpen{},
		&actionprocessors.ActionProcessorGamma{},
		&actionprocessors.ActionProcessorContrast{},
		&actionprocessors.ActionProcessorBrightness{},
		&actionprocessors.ActionProcessorBlur{},
		&actionprocessors.ActionProcessorExractColors{},
		&actionprocessors.ActionProcessorBase64{},
		&actionprocessors.ActionProcessorSave{},
	)
}

func (ic *Converter) RegisterImageProcessor(processors ...ImageProcessor) *Converter {
	if ic.processors == nil {
		ic.processors = map[string]ImageProcessor{}
	}
	for _, prc := range processors {
		ic.processors[prc.Name()] = prc
	}
	return ic
}

// Name of the converter
func (ic *Converter) Name() string {
	return "image"
}

// Test if action is suitable to perform
func (ic *Converter) Test(action *models.Action) bool {
	return ic.processors != nil && ic.processors[action.Name] != nil
}

// Process input file
func (ic *Converter) Process(in converters.Input, out converters.Output) error {
	if !in.Meta().Type.IsImage() {
		return ErrInvalidInputFile
	}
	var (
		action         = in.Action()
		contentType    = actionprocessors.ContentTypeFromExt(defStr(filepath.Ext(in.Task().Target), in.Meta().ObjectTypeExt()))
		quality        = int(action.ValueInt32(ActionParamJPEGQuality, 0))
		imgReader, err = imagereader.Decode(in.ObjectReader(), contentType, quality)
	)
	if err != nil {
		return errors.Wrap(errImageDecode, err.Error())
	}

	if processor := ic.processors[action.Name]; processor != nil {
		err = processor.Process(in, out, action, imgReader)
	} else {
		err = ErrUnsupportedAction
	}

	if err == nil && imgReader.Image() != nil {
		err = out.SetOutput(imgReader)
	}
	return err
}

func defStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
