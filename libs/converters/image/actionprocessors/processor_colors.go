package actionprocessors

import (
	"image"

	"github.com/EdlinOrg/prominentcolor"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

type ActionProcessorExractColors struct{}

func (ActionProcessorExractColors) Name() string { return ActionExtractColors }

func (ActionProcessorExractColors) Process(in converters.Input, out converters.Output, action *models.Action, imgReader ImageReader) error {
	v := action.ValueInt64(ActionParamValue, 0)
	if v < 1 {
		v = 3
	}
	colors, err := colorsExtraction(imgReader.Image(), int(v))
	if err != nil {
		return err
	}
	out.Meta().SetExt("colors", colors)
	return nil
}

func colorsExtraction(img image.Image, k int) ([]string, error) {
	resizeSize := uint(prominentcolor.DefaultSize)
	bgmasks := prominentcolor.GetDefaultMasks()
	bitattr := prominentcolor.ArgumentNoCropping
	// Skip error as it's not so important
	colors, _ := prominentcolor.KmeansWithAll(k, img, bitattr, resizeSize, bgmasks)

	var arr []string
	for _, color := range colors {
		arr = append(arr, "#"+color.AsString())
	}

	return arr, nil
}
