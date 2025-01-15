//
// @project apfs 2017 - 2020
// @author Dmitry Ponomarev <demdxx@gmail.com> 2017 - 2020
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

	apfs "github.com/apfs-io/apfs"
	"github.com/apfs-io/apfs/libs/converters/image"
	"github.com/apfs-io/apfs/libs/converters/procedure"
	"github.com/apfs-io/apfs/models"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := apfs.NewClient(ctx, os.Getenv("STORAGE_CONNECT"))
	fatalError(err, "storage")

	err = initImageStore(client)
	fatalError(err, "image storage")

	nobj, err := client.UploadFile(ctx, "images", "", "/testdata/crowd.jpg", []string{"test1"}, false)
	fatalError(err, "image upload")

	var obj *models.Object

	for i := 1; i < 10; i++ {
		obj, err = client.Head(ctx, apfs.NewObjectID(nobj.ID))
		fatalError(err, "image head")
		time.Sleep(time.Second)
		fmt.Println(i, ">", obj.Status.String(), obj.StatusMessage)
		if obj.Status.IsProcessed() {
			break
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(obj)
}

func initImageStore(storageClient apfs.Client) error {
	return storageClient.SetManifest(context.Background(), "images", &models.Manifest{
		Version:      "test-v1",
		ContentTypes: []string{"image/*"},
		Stages: []*models.ManifestTaskStage{
			{
				Name: "",
				Tasks: []*models.ManifestTask{
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

func resulution(name string, size int, extacts ...*models.Action) *models.ManifestTask {
	return &models.ManifestTask{
		Source: "@",
		Target: name,
		Type:   models.TypeImage,
		Actions: append([]*models.Action{
			procedure.NewActionAsFile("image-resize-w", "",
				false, strconv.Itoa(size), "{{inputFile}}", "{{outputFile}}"),
		}, extacts...),
	}
}

func fatalError(err error, msg ...any) {
	if err != nil {
		log.Fatalln(append([]any{err}, msg...))
	}
}
