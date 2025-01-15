package data

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMd5(t *testing.T) {
	hash, err := HashMd5([]byte(`test`))
	assert.Nil(t, err)
	assert.Equal(t, `098f6bcd4621d373cade4e832627b4f6`, hash)

	_, hash, err = HashDataMd5(bytes.NewBuffer([]byte(`test`)))
	assert.Nil(t, err)
	assert.Equal(t, `098f6bcd4621d373cade4e832627b4f6`, hash)
}
