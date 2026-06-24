//
// @project apfs 2017 - 2020, 2025
// @author Dmitry Ponomarev
//

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/apfs-io/apfs"
	"github.com/apfs-io/apfs/libs/client"
	"github.com/apfs-io/apfs/models"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apfsClient, err := apfs.Connect(ctx, os.Getenv("STORAGE_CONNECT"))
	fatalError(err, "storage connection failed")
	defer func() { _ = apfsClient.Close() }()

	imgClient := apfsClient.WithGroup("images")

	err = initImageStore(ctx, imgClient)
	fatalError(err, "failed to initialize image storage")

	newObj, err := imgClient.UploadFile(ctx, "/testdata/crowd.jpg", client.WithTags("test1"))
	fatalError(err, "image upload failed")

	var obj *apfs.Object
	fmt.Println("Uploaded image:", newObj.ID, newObj.Size, "bytes")

	for i := range 9 {
		obj, err = imgClient.Head(ctx, apfs.NewObjectID(newObj.ID), client.WithState())
		fatalError(err, "failed to retrieve image metadata")
		time.Sleep(time.Second)

		if i > 0 {
			fmt.Print("\033[1A\033[2K") // move cursor up and clear line
		}
		fmt.Printf("%d > %s %d/%d - Failed: %d\n",
			i+1, obj.Status.String(),
			obj.State.Counters.Succeeded,
			obj.State.Counters.Total,
			obj.State.Counters.Failed)
		if obj.Status.IsProcessed() {
			break
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(obj)
}

// initImageStore initializes the image storage with a workflow manifest.
func initImageStore(ctx context.Context, c apfs.Client) error {
	wf := &models.Workflow{
		Version:      "2",
		ContentTypes: []string{"image/*"},
		Jobs: map[string]*models.WorkflowJob{
			"card":       imageResizeJob("card", 600),
			"small":      imageResizeJob("small", 200),
			"hd":         imageResizeJob("hd", 1280),
			"full-hd":    imageResizeJob("full-hd", 1920),
			"2k":         imageResizeJob("2k", 2560),
			"4k":         imageResizeJob("4k", 3840),
			"5k":         imageResizeJob("5k", 5120),
			"b64preview": b64PreviewJob(),
		},
	}
	return c.SetWorkflow(ctx, wf)
}

// imageResizeJob creates a workflow job that resizes an image to a given width.
func imageResizeJob(name string, size int) *models.WorkflowJob {
	return &models.WorkflowJob{
		Steps: []*models.WorkflowStep{
			{
				Uses: "procedure/image-resize-w",
				With: map[string]any{
					"source":     "@",
					"target":     name,
					"type":       string(models.TypeImage),
					"width":      strconv.Itoa(size),
					"inputFile":  "{{inputFile}}",
					"outputFile": "{{outputFile}}",
				},
			},
		},
	}
}

func b64PreviewJob() *models.WorkflowJob {
	return &models.WorkflowJob{
		Steps: []*models.WorkflowStep{
			{Uses: "image/extract-colors", With: map[string]any{"count": 7}},
			{Uses: "image/fit", With: map[string]any{"width": 50, "height": 50, "filter": "lanczos"}},
			{Uses: "image/blur", With: map[string]any{"radius": 3}},
			{Uses: "image/b64-extract", With: map[string]any{"target": "b64preview"}},
		},
	}
}

func fatalError(err error, msg ...any) {
	if err != nil {
		log.Fatalln(append([]any{err}, msg...))
	}
}
