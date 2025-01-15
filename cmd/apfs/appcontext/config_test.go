package appcontext

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigStringify(t *testing.T) {
	assert.False(t, strings.HasPrefix(Config.String(), `{"error":"`))
}
