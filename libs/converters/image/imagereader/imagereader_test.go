package imagereader

import (
	"bytes"
	_ "image/gif"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

const gifCode = "\x47\x49\x46\x38\x39\x61\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00\x2c\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02\x44\x01\x00\x3b"

func TestImageDecode(t *testing.T) {
	imgReader, err := Decode(bytes.NewBuffer([]byte(gifCode)), "image/gif", 0)
	assert.NoError(t, err)
	assert.NotNil(t, imgReader.Image())

	data, err := ioutil.ReadAll(imgReader)
	assert.NoError(t, err, `read image data`)
	assert.True(t, len(gifCode) > len(data)-5 && len(gifCode) < len(data)+5)
	assert.NotNil(t, imgReader.Image())

	assert.NoError(t, imgReader.Close())
}
