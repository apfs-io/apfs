package procedure

import (
	"encoding/json"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

func TestProcedureProcess(t *testing.T) {
	var (
		_, fileName, _, _ = runtime.Caller(0)
		conv              = New(filepath.Join(filepath.Dir(fileName), "procedures"))
		action            = &models.Action{
			Name: ActionName,
			Values: map[string]any{
				ParamName:         "test.sh",
				ParamArguments:    []string{"arg1", "arg2"},
				ParamToJSONString: true,
				ParamTargetMeta:   "target",
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

	assert.NoError(t, err, `process "test.sh" command`)
	assert.Nil(t, output.ObjectReader(), `awaited response as JSON field`)
	assert.Equal(t, json.RawMessage(`"arg1 arg2\ntest\n"`), outMeta.GetExt("target"), `invalid target value`)
	assert.NoError(t, conv.Finish(input, output), `process "test.sh" command`)
}

func TestProcedureProcessFile(t *testing.T) {
	var (
		_, fileName, _, _ = runtime.Caller(0)
		conv              = New(filepath.Join(filepath.Dir(fileName), "procedures"))
		action            = &models.Action{
			Name: ActionName,
			Values: map[string]any{
				ParamName:      "test.sh",
				ParamArguments: []string{"arg1", "arg2"},
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

	assert.NoError(t, err, `process "test.sh" command`)
	assert.NotNil(t, output.ObjectReader(), `invalid output file`)

	data, _ := io.ReadAll(output.ObjectReader())
	assert.Equal(t, []byte("arg1 arg2\ntest\n"), data, `invalid target value`)
	assert.NoError(t, conv.Finish(input, output), `process "test.sh" command`)
}
