package actionprocessors

import (
	"encoding/base64"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/libs/converters/image/imagereader"
	"github.com/apfs-io/apfs/models"
)

type ActionProcessorBase64 struct{}

func (ActionProcessorBase64) Name() string { return ActionBase64 }

func (ActionProcessorBase64) Process(in converters.Input, out converters.Output, action *models.Action, imgReader ImageReader) error {
	var (
		quality          = int(action.ValueInt32(ActionParamJPEGQuality, 0))
		finalContentType = ContentTypeFromExt(defStr(filepath.Ext(in.Task().Target), in.Meta().ObjectTypeExt()))
		b64ContentType   = action.ValueString(ActionParamB64Format, finalContentType)
		target           = action.ValueString(ActionParamMetaField, "b64data")
		b64ImageReader   = imagereader.NewImageReader(imgReader.Image(), b64ContentType, quality)
	)
	out.Meta().SetExt(target, encodeB64Reader(b64ImageReader, b64ContentType))
	return nil
}

func encodeB64Reader(reader io.Reader, contentType string) string {
	data, _ := ioutil.ReadAll(reader)
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func defStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
