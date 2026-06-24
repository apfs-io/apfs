package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const videoWorkflowYAML = `
version: "2"
name: "Video pipeline"
keep_original: true
content_types:
  - "video/*"
validate:
  max_size: 2GB
  content_types: ["video/*"]
jobs:
  probe:
    runs-on: any
    steps:
      - name: Extract metadata
        uses: video.probe
        with:
          target-meta: codec,duration
  transcode-main:
    runs-on: large
    needs: [probe]
    on-failure: retry:3
    steps:
      - name: Transcode H.264
        uses: video.transcode
        with:
          codec: h264
          target: main.mp4
  thumbnails:
    runs-on: small
    needs: [probe]
    on-failure: continue
    if: "${{ probe.outputs.duration < 3600 }}"
    steps:
      - name: Generate thumbs
        uses: video.thumbnails
        with:
          count: 10
`

func TestParseWorkflowYAML(t *testing.T) {
	w, err := ParseWorkflow([]byte(videoWorkflowYAML))
	require.NoError(t, err)
	require.NotNil(t, w)

	assert.Equal(t, "2", w.Version)
	assert.Equal(t, "Video pipeline", w.Name)
	assert.True(t, w.ShouldKeepOriginal())
	assert.Equal(t, []string{"video/*"}, w.ContentTypes)
	assert.NotNil(t, w.Validate)
	assert.Equal(t, "2GB", w.Validate.MaxSize)
	assert.Equal(t, int64(2*1024*1024*1024), w.Validate.MaxSizeBytes())

	assert.Equal(t, 3, len(w.Jobs))
	probe := w.Jobs["probe"]
	require.NotNil(t, probe)
	assert.Equal(t, "any", probe.RunsOn)
	assert.Len(t, probe.Steps, 1)
	assert.Equal(t, "video.probe", probe.Steps[0].Uses)

	transcode := w.Jobs["transcode-main"]
	require.NotNil(t, transcode)
	assert.Equal(t, []string{"probe"}, transcode.Needs)
	assert.Equal(t, "retry:3", transcode.OnFailure)

	thumbs := w.Jobs["thumbnails"]
	require.NotNil(t, thumbs)
	assert.Equal(t, "continue", thumbs.OnFailure)
	assert.Equal(t, "${{ probe.outputs.duration < 3600 }}", thumbs.If)
}

func TestParseWorkflowJSON(t *testing.T) {
	jsonData := `{
		"version":"2",
		"name":"Image pipeline",
		"jobs":{
			"resize": {
				"runs_on": "any",
				"steps":[{"name":"Resize","uses":"image.fit","with":{"width":300}}]
			}
		}
	}`
	w, err := ParseWorkflow([]byte(jsonData))
	require.NoError(t, err)
	assert.Equal(t, "2", w.Version)
	assert.Equal(t, 1, len(w.Jobs))
	assert.Equal(t, "image.fit", w.Jobs["resize"].Steps[0].Uses)
}

func TestParseWorkflowLegacyJSON(t *testing.T) {
	// v1-format JSON (stages/tasks) is now parsed as a raw Workflow.
	// The parser no longer auto-converts legacy manifests; callers that need
	// v1 support should use models.FromLegacyManifest explicitly.
	legacyJSON := `{
		"version": "1",
		"stages": []
	}`
	w, err := ParseWorkflow([]byte(legacyJSON))
	require.NoError(t, err)
	require.NotNil(t, w)
	assert.Equal(t, "1", w.Version) // version is kept as-is
	assert.Empty(t, w.Jobs)         // no jobs — v1 stages are not auto-converted
}

func TestParseWorkflowEmpty(t *testing.T) {
	w, err := ParseWorkflow(nil)
	require.NoError(t, err)
	assert.True(t, w.IsEmpty())
}
