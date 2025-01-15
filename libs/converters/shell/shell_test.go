package shell

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

func TestShellProcess(t *testing.T) {
	var (
		conv   = Converter{}
		action = &models.Action{
			Name: ActionName,
			Values: map[string]any{
				ParamCommand:      "head",
				ParamTargetMeta:   "target",
				ParamToJSONString: "1",
			},
		}
		inMeta       = &models.ItemMeta{}
		outMeta      = &models.ItemMeta{}
		manifestTask = &models.ManifestTask{}
	)

	assert.True(t, conv.Test(action), `action test`)

	input := converters.NewInput(strings.NewReader("test"), manifestTask, action, inMeta)
	output := converters.NewOutput(outMeta)
	err := conv.Process(input, output)

	assert.NoError(t, err, `process "head" command`)
	assert.Nil(t, output.ObjectReader(), `awaited response as JSON field`)
	assert.Equal(t, json.RawMessage(`"test"`), outMeta.GetExt("target"), `invalid target value`)
	assert.NoError(t, conv.Finish(input, output), `process "head" command`)
}

func TestShellProcessFile(t *testing.T) {
	var (
		conv   = Converter{}
		action = &models.Action{
			Name: ActionName,
			Values: map[string]any{
				ParamCommand: "head",
			},
		}
		inMeta       = &models.ItemMeta{}
		outMeta      = &models.ItemMeta{}
		manifestTask = &models.ManifestTask{}
	)

	assert.True(t, conv.Test(action), `action test`)

	input := converters.NewInput(strings.NewReader("test"), manifestTask, action, inMeta)
	output := converters.NewOutput(outMeta)
	err := conv.Process(input, output)

	if assert.NoError(t, err, `process "head" command`) {
		if assert.NotNil(t, output.ObjectReader(), `invalid output file`) {
			data, _ := io.ReadAll(output.ObjectReader())
			assert.Equal(t, []byte(`test`), data, `invalid target value`)
			assert.NoError(t, conv.Finish(input, output), `process "head" command`)
		}
	}
}
