package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apfs-io/apfs/models"
)

func TestWorkflowRoundTripPreservesRunAndDocker(t *testing.T) {
	keep := true
	orig := &models.Workflow{
		Version:      "3",
		Name:         "video",
		Description:  "encode",
		ContentTypes: []string{"video/*"},
		KeepOriginal: &keep,
		Jobs: map[string]*models.WorkflowJob{
			"preview": {
				RunsOn:    "any",
				OnFailure: "continue",
				Steps: []*models.WorkflowStep{
					{
						Name: "first frame",
						Uses: "docker",
						Run:  "ffmpeg -i {{.input}} -vframes 1 {{.target}}",
						With: map[string]any{"target": "preview.jpg"},
						Docker: &models.WorkflowStepDocker{
							Image:           "jrottenberg/ffmpeg:4.4-alpine",
							PullImage:       true,
							RetainContainer: false,
							RemoveAfterDone: true,
							ContainerName:   "apfs-ffmpeg",
						},
					},
				},
			},
		},
	}

	proto := WorkflowFromModel(orig)
	require.NotNil(t, proto)
	require.Len(t, proto.Jobs, 1)
	step := proto.Jobs[0].Steps[0]
	assert.Equal(t, "ffmpeg -i {{.input}} -vframes 1 {{.target}}", step.GetRun())
	require.NotNil(t, step.GetDocker())
	assert.Equal(t, "jrottenberg/ffmpeg:4.4-alpine", step.GetDocker().GetImage())
	assert.True(t, step.GetDocker().GetPullImage())
	assert.True(t, step.GetDocker().GetRemoveAfterDone())
	assert.Equal(t, "apfs-ffmpeg", step.GetDocker().GetContainerName())

	back := WorkflowToModel(proto)
	require.NotNil(t, back)
	job := back.Jobs["preview"]
	require.NotNil(t, job)
	require.Len(t, job.Steps, 1)
	assert.Equal(t, orig.Jobs["preview"].Steps[0].Run, job.Steps[0].Run)
	require.NotNil(t, job.Steps[0].Docker)
	assert.Equal(t, orig.Jobs["preview"].Steps[0].Docker.Image, job.Steps[0].Docker.Image)
	assert.Equal(t, orig.Jobs["preview"].Steps[0].Docker.PullImage, job.Steps[0].Docker.PullImage)
	assert.Equal(t, orig.Jobs["preview"].Steps[0].Docker.RemoveAfterDone, job.Steps[0].Docker.RemoveAfterDone)
	assert.Equal(t, orig.Jobs["preview"].Steps[0].Docker.ContainerName, job.Steps[0].Docker.ContainerName)
	assert.Equal(t, "preview.jpg", job.Steps[0].With["target"])
}
