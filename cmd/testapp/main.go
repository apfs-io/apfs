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
	"github.com/apfs-io/apfs/libs/converters/image"
	"github.com/apfs-io/apfs/libs/converters/procedure"
	"github.com/apfs-io/apfs/models"
)

func main() {
	// Create a cancelable context for the application lifecycle.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to the APFS storage using the connection string from the environment variable.
	apfsClient, err := apfs.Connect(ctx, os.Getenv("STORAGE_CONNECT"))
	fatalError(err, "storage connection failed")
	defer apfsClient.Close()

	// Create a client for the "images" group.
	imgClient := apfsClient.WithGroup("images")

	// Initialize the image store with the required manifest.
	err = initImageStore(ctx, imgClient)
	fatalError(err, "failed to initialize image storage")

	// Upload an image file to the storage.
	newObj, err := imgClient.UploadFile(ctx, "/testdata/crowd.jpg", client.WithTags("test1"))
	fatalError(err, "image upload failed")

	// Poll the status of the uploaded image until it is processed.
	var objMeta *models.Object
	for i := range 9 {
		objMeta, err = imgClient.Head(ctx, apfs.NewObjectID(newObj.ID))
		fatalError(err, "failed to retrieve image metadata")
		time.Sleep(time.Second)
		fmt.Println(i+1, ">", objMeta.Status.String(), objMeta.StatusMessage)
		if objMeta.Status.IsProcessed() {
			break
		}
	}

	// Output the final object metadata in JSON format.
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(objMeta)
}

// initImageStore initializes the image storage with a manifest defining processing stages and tasks.
func initImageStore(ctx context.Context, client apfs.Client) error {
	return client.SetManifest(ctx, &models.Manifest{
		Version:      "test-v1",
		ContentTypes: []string{"image/*"},
		Stages: []*models.ManifestTaskStage{
			{
				Name: "",
				Tasks: []*models.ManifestTask{
					// Define tasks for resizing images to various resolutions.
					resulution("card", 600),
					resulution("small", 200),
					resulution("hd", 1280),
					resulution("full-hd", 1920),
					resulution("2k", 2560),
					resulution("4k", 3840),
					resulution("5k", 5120),
					{
						Source: "@",
						Type:   models.TypeImage,
						Actions: []*models.Action{
							// Extract colors, resize, blur, and save the image.
							image.NewActionExtractColors(7),
							image.NewActionFit(50, 50, "lanczos"),
							image.NewActionBlur(3),
							image.NewActionB64Extract("", "b64preview"),
							image.NewActionSave(false),
						},
					},
				},
			},
		},
	})
}

// resulution creates a task for resizing an image to a specific resolution.
func resulution(name string, size int, extacts ...*models.Action) *models.ManifestTask {
	return &models.ManifestTask{
		Source: "@",
		Target: name,
		Type:   models.TypeImage,
		Actions: append([]*models.Action{
			// Use a procedure to resize the image.
			procedure.NewActionAsFile("image-resize-w", "",
				false, strconv.Itoa(size), "{{inputFile}}", "{{outputFile}}"),
		}, extacts...),
	}
}

// fatalError logs and exits the program if an error occurs.
func fatalError(err error, msg ...any) {
	if err != nil {
		log.Fatalln(append([]any{err}, msg...))
	}
}
