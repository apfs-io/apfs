package storerrors

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNotFound(t *testing.T) {
	err1 := os.ErrNotExist
	err2 := WrapNotFound(`1234`, err1)
	assert.False(t, IsNotFound(err1))
	assert.False(t, IsNotFound((*NotFound)(nil)))
	assert.True(t, IsNotFound(err2))
	assert.True(t, IsNotFound(errors.Wrap(err2, `wrapped`)))
	assert.True(t, len(WrapNotFound(``, err1).Error()) > 0)
	assert.True(t, len(WrapNotFound(`1234`, err1).Error()) > 0)
	assert.True(t, len(WrapNotFound(`1234`, nil).Error()) > 0)
	assert.EqualError(t, err2.Unwrap(), err1.Error())
}
