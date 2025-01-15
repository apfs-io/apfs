package actionprocessors

// Task Action list
const (
	ActionValidateSize  = "image.validate-size"
	ActionResize        = "image.resize"
	ActionFit           = "image.fit"
	ActionFill          = "image.fill"
	ActionBlur          = "image.blur"
	ActionSharpen       = "image.sharpen"
	ActionGamma         = "image.gamma"
	ActionContrast      = "image.contrast"
	ActionBrightness    = "image.brightness"
	ActionExtractColors = "image.extract-colors"
	ActionBase64        = "image.base64"
	ActionSave          = "image.save"
)

// Action params...
const (
	ActionParamValue       = "value"
	ActionParamWidth       = "width"
	ActionParamHeight      = "height"
	ActionParamMaxWidth    = "max-width"
	ActionParamMaxHeight   = "max-height"
	ActionParamAnchor      = "anchor"
	ActionParamFilter      = "filter"
	ActionParamB64Format   = "format"
	ActionParamMetaField   = "target-meta"
	ActionParamSave        = "save"
	ActionParamJPEGQuality = "jpeg.quality"
)
