package data

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupReader(t *testing.T) {
	reader := NewGroupReader(strings.NewReader(`test`), strings.NewReader(`file`))
	defer reader.Close()
	data, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, []byte(`testfile`), data)
}
