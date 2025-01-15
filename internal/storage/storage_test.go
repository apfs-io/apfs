package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/apfs-io/apfs/internal/driver/fs"
	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor/memory"
	"github.com/apfs-io/apfs/libs/converters/image"
	"github.com/apfs-io/apfs/libs/converters/procedure"
	"github.com/apfs-io/apfs/models"
)

var (
	testStorePath    = "teststore"
	proceduresDir, _ = filepath.Abs("../../deploy/procedures")
	fsdriver, _      = fs.NewStorage(testStorePath)
	storage          = NewStorage(
		WithDatabase(&DatabaseMock{}),
		WithDriver(fsdriver),
		WithProcessingStatus(&memory.KVMemory{}),
		WithConverters(image.NewDefaultConverter(),
			procedure.New(proceduresDir)),
	)
)

func TestStorageUpload(t *testing.T) {
	var (
		ctx, cancel = context.WithTimeout(context.TODO(), time.Second*10)
		obj         npio.Object
		err         error
	)
	defer cancel()

	t.Run("file-upload", func(t *testing.T) {
		if obj, err = storage.UploadFile(ctx, "images", "teststore/bucket/file/original.jpg"); err != nil {
			t.Error(err)
		}
	})

	t.Run("file-delete", func(t *testing.T) {
		if err = storage.Delete(ctx, obj); err != nil {
			t.Error(err)
		}
	})

	os.RemoveAll(filepath.Join(testStorePath, "images"))
}

func TestStorageProcess(t *testing.T) {
	var (
		object      npio.Object
		err         error
		tags        = []string{"tag1", "tag2"}
		ctx, cancel = context.WithTimeout(context.TODO(), time.Second*10)
		manifest    = &models.Manifest{
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
	manifest.PrepareInfo()

	t.Run("set-images-manifest", func(t *testing.T) {
		assert.NoError(t, storage.SetManifest(ctx, "images", manifest), "update images manifest")
	})

	t.Run("object-upload", func(t *testing.T) {
		object, err = storage.UploadFile(ctx,
			"images", "teststore/bucket/file/original.jpg",
			WithTags(tags),
			WithParams(nil),
		)
		assert.NoError(t, err, "upload new file")
	})

	t.Run("object-process", func(t *testing.T) {
		complete, err := storage.ProcessTasks(ctx, object, AllTasks, AllStages)
		assert.NoError(t, err, "task processing error")
		assert.True(t, complete, "processing must be completed")
	})

	t.Run("object-adjust-process", func(t *testing.T) {
		object, err := storage.Open(ctx, object.ID().String())
		assert.NoError(t, err, "open object")
		assert.NotNil(t, object, "object reference")

		object.Manifest().Stages[0].Tasks = object.Manifest().Stages[0].Tasks[1:]
		items := object.Meta().ExcessItems(object.Manifest())
		if assert.Equal(t, 1, len(items)) {
			removeSubObjects := make([]string, 0, len(items))
			for _, it := range items {
				removeSubObjects = append(removeSubObjects, it.Fullname())
			}
			err := storage.Delete(ctx, object, removeSubObjects...)
			assert.NoError(t, err, "remove extra objects")
			assert.Equal(t, 3, len(object.Meta().Items))
		}
	})

	t.Run("object-delete", func(t *testing.T) {
		err := storage.Delete(ctx, object)
		assert.NoError(t, err)
	})

	os.RemoveAll(filepath.Join(testStorePath, "images"))
}
