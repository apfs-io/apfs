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
	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor/memory"
	"github.com/apfs-io/apfs/internal/storage/processor"
	"github.com/apfs-io/apfs/libs/converters/image"
	"github.com/apfs-io/apfs/libs/converters/procedure"
	"github.com/apfs-io/apfs/models"
)

var (
	testStorePath    = "teststore"
	proceduresDir, _ = filepath.Abs("../../deploy/procedures")
	fsdriver         *fs.Storage
	storage          *Storage
	procc            *processor.Processor
)

func init() {
	_, filePath, _, _ := runtime.Caller(0)
	__dir, _ := filepath.Abs(filepath.Dir(filePath))
	testStorePath = filepath.Join(__dir, "teststore")
	proceduresDir, _ = filepath.Abs(filepath.Join(__dir, "../../deploy/procedures"))

	// Init file system driver for storage
	fsdriver, _ = fs.NewStorage(testStorePath)

	// Init state KV driver
	processingState := &memory.KVMemory{}

	// Init storage of the file objects
	storage = NewStorage(
		WithDatabase(&DatabaseMock{}),
		WithDriver(fsdriver),
		WithProcessingStatus(processingState),
	)

	// Init object processor
	procc, _ = processor.NewProcessor(
		processor.WithConverters(image.NewDefaultConverter(),
			procedure.New(proceduresDir)),
		processor.WithStorage(storage),
		processor.WithDriver(fsdriver),
		processor.WithProcessingStatus(processingState),
		processor.WithMaxRetries(3),
	)
}

func TestStorageUpload(t *testing.T) {
	var (
		ctx, cancel = context.WithTimeout(context.TODO(), time.Second*10)
		obj         npio.Object
		err         error
	)
	defer cancel()

	obj, err = storage.UploadFile(ctx, "images",
		filepath.Join(testStorePath, "bucket/file/original.jpg"))

	if assert.NoError(t, err, "upload file") {
		assert.NoError(t, storage.Delete(ctx, obj), "delete object")
	}

	_ = os.RemoveAll(filepath.Join(testStorePath, "images"))
}

func TestStorageProcess(t *testing.T) {
	const (
		imagesBucket = "images"
	)

	var (
		object           npio.Object
		tags             = []string{"tag1", "tag2"}
		originalFilepath = filepath.Join(testStorePath, "bucket/file/original.jpg")
		ctx, cancel      = context.WithTimeout(context.TODO(), time.Second*10)
		manifest         = &models.Manifest{
			Stages: []*models.ManifestTaskStage{
				{
					Name: "",
					Tasks: []*models.ManifestTask{
						{
							Source: "@",
							Target: "icon.png",
							Actions: []*models.Action{
								image.NewActionValidateSize(100, 100, 1000, 1000),
								image.NewActionFill(100, 100, "center", "linear"),
								image.NewActionGamma(3),
								image.NewActionExtractColors(3),
							},
							Required: true,
						},
						{
							Source: "@",
							Target: "blur.png",
							Actions: []*models.Action{
								image.NewActionValidateSize(150, 150, 1000, 1000),
								image.NewActionResize(150, 150, "linear"),
								image.NewActionBlur(3),
								image.NewActionExtractColors(2),
								image.NewActionB64Extract("", "b64data-test"),
							},
							Required: true,
						},
						{
							Source: "@",
							Target: "skip.png",
							Actions: []*models.Action{
								image.NewActionValidateSize(100, 100, 300, 300),
								image.NewActionFill(100, 100, "center", "linear"),
								image.NewActionGamma(3),
								image.NewActionExtractColors(3),
							},
							Required: false,
						},
						{
							Source: "blur.png",
							Actions: []*models.Action{
								image.NewActionB64Extract("", "b64data-test2"),
							},
							Required: true,
						},
					},
				},
				{
					Name: "extra",
					Tasks: []*models.ManifestTask{
						{
							Source: "@",
							Type:   models.TypeImage,
							Actions: []*models.Action{
								image.NewActionFit(50, 50, "lanczos"),
								image.NewActionBlur(3),
								image.NewActionB64Extract("", "b64data"),
								image.NewActionSave(false),
							},
							Required: true,
						},
						{
							Source: "blur.png",
							Target: "blur.jpeg",
							Actions: []*models.Action{
								image.NewActionSave(true),
							},
							Required: false,
						},
					},
				},
				// {
				// 	Name: "object-meta",
				// 	Tasks: []*models.ManifestTask{
				// 		{
				// 			Source: "@",
				// 			Type:   models.TypeImage,
				// 			Actions: []*models.Action{
				// 				procedure.NewActionMeta("face_meta.py", "faces", false, "{{inputFile}}"),
				// 			},
				// 		},
				// 		{
				// 			Source: "@",
				// 			Type:   models.TypeImage,
				// 			Actions: []*models.Action{
				// 				procedure.NewActionMeta("object-detection2.py", "objects", false, "{{inputFile}}"),
				// 			},
				// 		},
				// 	},
				// },
			},
		}
	)
	defer cancel()

	// Prepare manifest info
	manifest.PrepareInfo()

	// 1. Set images manifest
	err := storage.SetManifest(ctx, imagesBucket, manifest)
	if !assert.NoError(t, err, "Set images manifest") {
		return
	}

	// 2. Upload new file to images bucket
	object, err = storage.UploadFile(ctx, imagesBucket,
		originalFilepath, WithTags(tags), WithParams(nil))
	if !assert.NoError(t, err, "upload new file: "+originalFilepath) {
		return
	}

	// 3. Process uploaded object with all tasks and stages
	complete, err := procc.ProcessTasks(ctx, object, AllTasks, AllStages)
	if !assert.NoError(t, err, "task processing error") {
		return
	}
	if !assert.True(t, complete, "processing must be completed") {
		return
	}

	// 4. Adjust object with extra tasks
	objectAdjust, err := storage.Object(ctx, object.ID().String())
	if assert.NoError(t, err, "open object") && assert.NotNil(t, object, "object reference") {
		manifest := objectAdjust.Manifest()
		manifest.Stages[0].Tasks = manifest.Stages[0].Tasks[1:]

		items := objectAdjust.Meta().ExcessItems(manifest)
		if assert.Equal(t, 1, len(items)) {
			removeSubObjects := make([]string, 0, len(items))
			for _, it := range items {
				removeSubObjects = append(removeSubObjects, it.Fullname())
			}
			err := storage.Delete(ctx, objectAdjust, removeSubObjects...)
			assert.NoError(t, err, "remove extra objects")
			assert.Equal(t, 3, len(objectAdjust.Meta().Items))
		}
	}

	// 5. Delete uploaded object from storage
	assert.NoError(t, storage.Delete(ctx, object), "delete object")

	// Finaly remove all files
	_ = os.RemoveAll(filepath.Join(testStorePath, imagesBucket))
}
