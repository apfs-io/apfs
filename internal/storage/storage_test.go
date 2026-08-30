package storage

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/apfs-io/apfs/internal/driver/fs"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor/memory"
	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/internal/workflow"
	"github.com/apfs-io/apfs/libs/converters/image"
	"github.com/apfs-io/apfs/models"
)

var (
	testStorePath = "teststore"
	fsdriver      *fs.Storage
	storage       *Storage
	wfExec        *workflow.Executor
)

func init() {
	_, filePath, _, _ := runtime.Caller(0)
	__dir, _ := filepath.Abs(filepath.Dir(filePath))
	testStorePath = filepath.Join(__dir, "teststore")

	fsdriver, _ = fs.NewStorage(testStorePath)

	processingState := &memory.KVMemory{}

	storage = NewStorage(
		WithDatabase(&DatabaseMock{}),
		WithDriver(fsdriver),
		WithProcessingStatus(processingState),
	)

	reg := workflow.NewRunnerRegistry()
	reg.Register(image.NewDefaultConverter().StepRunner())
	wfExec = workflow.NewExecutor(NewWorkflowStorage(storage), reg)
}

func TestStorageUpload(t *testing.T) {
	var (
		ctx, cancel = context.WithTimeout(context.TODO(), time.Second*10)
		obj         storio.Object
		err         error
	)
	defer cancel()

	obj, err = storage.UploadFile(ctx, "images",
		filepath.Join(testStorePath, "bucket/file/prim.jpg"))

	if assert.NoError(t, err, "upload file") {
		assert.NoError(t, storage.Delete(ctx, obj), "delete object")
	}

	_ = os.RemoveAll(filepath.Join(testStorePath, "images"))
}

func TestStorageProcess(t *testing.T) {
	const imagesBucket = "images"

	var (
		object           storio.Object
		tags             = []string{"tag1", "tag2"}
		originalFilepath = filepath.Join(testStorePath, "bucket/file/prim.jpg")
		ctx, cancel      = context.WithTimeout(context.TODO(), time.Second*10)
		wf               = &models.Workflow{
			Version: "2",
			Jobs: map[string]*models.WorkflowJob{
				"icon": {
					OnFailure: "fail",
					Steps: pipelineSteps("@", "icon.png",
						image.NewActionValidateSize(100, 100, 1000, 1000),
						image.NewActionFill(100, 100, "center", "linear"),
						image.NewActionGamma(3),
						image.NewActionExtractColors(3),
					),
				},
				"blur": {
					OnFailure: "fail",
					Steps: pipelineSteps("@", "blur.png",
						image.NewActionValidateSize(150, 150, 1000, 1000),
						image.NewActionResize(150, 150, "linear"),
						image.NewActionBlur(3),
						image.NewActionExtractColors(2),
						image.NewActionB64Extract("", "b64data-test"),
					),
				},
				"blur_b64": {
					OnFailure: "fail",
					Needs:     []string{"blur"},
					Steps: pipelineSteps("blur.png", "",
						image.NewActionB64Extract("", "b64data-test2"),
					),
				},
				"preview": {
					OnFailure: "fail",
					Steps: pipelineSteps("@", "",
						image.NewActionFit(50, 50, "lanczos"),
						image.NewActionBlur(3),
						image.NewActionB64Extract("", "b64data"),
						image.NewActionSave(false),
					),
				},
				"blur_jpeg": {
					OnFailure: "continue",
					Needs:     []string{"blur"},
					Steps: pipelineSteps("blur.png", "blur.jpeg",
						image.NewActionSave(true),
					),
				},
			},
		}
	)
	defer cancel()

	err := storage.SetWorkflow(ctx, imagesBucket, wf)
	if !assert.NoError(t, err, "Set images workflow") {
		return
	}

	object, err = storage.UploadFile(ctx, imagesBucket,
		originalFilepath, WithTags(tags), WithParams(nil))
	if !assert.NoError(t, err, "upload new file: "+originalFilepath) {
		return
	}

	complete, err := wfExec.ProcessObject(ctx, storage.ObjectWorkflow(ctx, object), object.ID().String(), nil, 0)
	if !assert.NoError(t, err, "task processing error") {
		return
	}
	if !assert.True(t, complete, "processing must be completed") {
		return
	}

	objectAdjust, err := storage.Object(ctx, object.ID().String())
	if assert.NoError(t, err, "open object") && assert.NotNil(t, object, "object reference") {
		reducedWF := &models.Workflow{
			Jobs: make(map[string]*models.WorkflowJob),
		}
		loaded := objectAdjust.Workflow()

		for jobID, job := range loaded.Jobs {
			isIconJob := false
			for _, step := range job.Steps {
				if tgt, _ := step.With["target"].(string); tgt == "icon.png" {
					isIconJob = true
					break
				}
			}
			if !isIconJob {
				reducedWF.Jobs[jobID] = job
			}
		}

		items := objectAdjust.Meta().ExcessItems(reducedWF)
		if assert.Equal(t, 1, len(items), "icon.png should be excess") {
			removeSubObjects := make([]string, 0, len(items))
			for _, it := range items {
				removeSubObjects = append(removeSubObjects, it.Fullname())
			}
			err := storage.Delete(ctx, objectAdjust, removeSubObjects...)
			assert.NoError(t, err, "remove extra objects")
			assert.Equal(t, 2, len(objectAdjust.Meta().Items), "blur.png and blur.jpeg remain")
		}
	}

	assert.NoError(t, storage.Delete(ctx, object), "delete object")

	_ = os.RemoveAll(filepath.Join(testStorePath, imagesBucket))
}

func pipelineSteps(source, target string, actions ...*models.Action) []*models.WorkflowStep {
	steps := make([]*models.WorkflowStep, 0, len(actions))
	for i, a := range actions {
		with := cloneActionValues(a)
		src := source
		if i > 0 && target != "" {
			src = target
		}
		if src != "" {
			with["source"] = src
		}
		if target != "" {
			with["target"] = target
		}
		steps = append(steps, &models.WorkflowStep{
			Name: a.Name,
			Uses: a.Name,
			With: with,
		})
	}
	return steps
}

func cloneActionValues(a *models.Action) map[string]any {
	if a == nil || len(a.Values) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(a.Values)+2)
	for k, v := range a.Values {
		out[k] = v
	}
	return out
}
