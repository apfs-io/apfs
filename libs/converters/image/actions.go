package image

import (
	"errors"

	"github.com/apfs-io/apfs/libs/converters/image/actionprocessors"
	"github.com/apfs-io/apfs/models"
)

// Task Action list
const (
	ActionValidateSize  = actionprocessors.ActionValidateSize
	ActionResize        = actionprocessors.ActionResize
	ActionFit           = actionprocessors.ActionFit
	ActionFill          = actionprocessors.ActionFill
	ActionBlur          = actionprocessors.ActionBlur
	ActionSharpen       = actionprocessors.ActionSharpen
	ActionGamma         = actionprocessors.ActionGamma
	ActionContrast      = actionprocessors.ActionContrast
	ActionBrightness    = actionprocessors.ActionBrightness
	ActionExtractColors = actionprocessors.ActionExtractColors
	ActionBase64        = actionprocessors.ActionBase64
	ActionSave          = actionprocessors.ActionSave
)

// Action params...
const (
	ActionParamValue       = actionprocessors.ActionParamValue
	ActionParamWidth       = actionprocessors.ActionParamWidth
	ActionParamHeight      = actionprocessors.ActionParamHeight
	ActionParamMaxWidth    = actionprocessors.ActionParamMaxWidth
	ActionParamMaxHeight   = actionprocessors.ActionParamMaxHeight
	ActionParamAnchor      = actionprocessors.ActionParamAnchor
	ActionParamFilter      = actionprocessors.ActionParamFilter
	ActionParamB64Format   = actionprocessors.ActionParamB64Format
	ActionParamMetaField   = actionprocessors.ActionParamMetaField
	ActionParamSave        = actionprocessors.ActionParamSave
	ActionParamJPEGQuality = actionprocessors.ActionParamJPEGQuality
)

// Error list...
var (
	ErrUnsupportedAction = errors.New("[image] unsupported action")
	ErrInvalidInputFile  = errors.New("[image] invalid input file")
)

// NewActionValidateSize with width and heigth
func NewActionValidateSize(width, height, maxWidth, maxHeight int) *models.Action {
	return models.NewAction(ActionValidateSize,
		ActionParamWidth, width,
		ActionParamHeight, height,
		ActionParamMaxWidth, maxWidth,
		ActionParamMaxHeight, maxHeight,
	)
}

// NewActionResize with width and heigth
func NewActionResize(width, height int, filter string) *models.Action {
	return models.NewAction(ActionResize,
		ActionParamWidth, width,
		ActionParamHeight, height,
		ActionParamFilter, filter)
}

// NewActionFit with width and heigth
func NewActionFit(width, height int, filter string) *models.Action {
	return models.NewAction(ActionFit,
		ActionParamWidth, width,
		ActionParamHeight, height,
		ActionParamFilter, filter)
}

// NewActionFill with width and heigth
func NewActionFill(width, height int, anchor, filter string) *models.Action {
	return models.NewAction(ActionFill,
		ActionParamWidth, width,
		ActionParamHeight, height,
		ActionParamAnchor, anchor,
		ActionParamFilter, filter)
}

// NewActionBlur with value
func NewActionBlur(value float64) *models.Action {
	return models.NewAction(ActionBlur, ActionParamValue, value)
}

// NewActionSharpen with value
func NewActionSharpen(value float64) *models.Action {
	return models.NewAction(ActionSharpen, ActionParamValue, value)
}

// NewActionGamma with value
func NewActionGamma(value float64) *models.Action {
	return models.NewAction(ActionGamma, ActionParamValue, value)
}

// NewActionContrast with value
func NewActionContrast(value float64) *models.Action {
	return models.NewAction(ActionContrast, ActionParamValue, value)
}

// NewActionBrightness with value
func NewActionBrightness(value float64) *models.Action {
	return models.NewAction(ActionBrightness, ActionParamValue, value)
}

// NewActionExtractColors with value
func NewActionExtractColors(value int64) *models.Action {
	return models.NewAction(ActionExtractColors, ActionParamValue, value)
}

// NewActionB64Extract with target meta field
func NewActionB64Extract(contentType, targetMeta string) *models.Action {
	return models.NewAction(ActionBase64,
		ActionParamB64Format, contentType,
		ActionParamMetaField, targetMeta,
	)
}

// NewActionSave object into the pipeline
// This action must be the last in the sequance, otherwhise it could be a problem
func NewActionSave(saves ...bool) *models.Action {
	save := len(saves) == 0 || saves[0]
	return models.NewAction(ActionSave, ActionParamSave, save)
}
