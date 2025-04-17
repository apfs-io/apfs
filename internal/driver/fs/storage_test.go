package fs

import (
	"context"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"testing"
	"time"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/models"
)

var (
	storageDir        = "../teststore"
	diskCollection, _ = NewStorage(storageDir)
)

func TestDiskCollectionCreate(t *testing.T) {
	var (
		testGroup   = "test"
		srcfile     = storageDir + "/bucket/file/prim.jpg"
		tags        = []string{"images", "test"}
		ctx, cancel = context.WithTimeout(context.TODO(), time.Second*10)
		file, err   = diskCollection.Create(ctx, testGroup, nil, false, url.Values{"tags": tags})
	)
	defer cancel()

	if err != nil {
		t.Error(err)
		return
	}

	file2, err := os.Open(srcfile)
	if err != nil {
		t.Error(err)
		return
	}
	defer file2.Close()

	if err = diskCollection.Update(ctx, file, models.OriginalFilename, file2, nil); err != nil {
		t.Error(err)
		return
	}

	if err = diskCollection.Clean(ctx, file); err != nil {
		t.Error(err)
		return
	}

	if err = diskCollection.Remove(ctx, file); err != nil {
		t.Error(err)
		return
	}

	os.RemoveAll(storageDir + "/" + testGroup)
}

func TestDiskCollectionOpen(t *testing.T) {
	file, err := diskCollection.Open(context.TODO(), npio.ObjectIDType("bucket/file"))
	if err != nil {
		t.Error(err)
		return
	}

	if file.Path() != "file" {
		t.Error("invalid codename")
	}
}
