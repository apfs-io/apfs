package fs

import (
	"context"
	_ "image/gif"  // Register GIF format
	_ "image/jpeg" // Register JPEG format
	_ "image/png"  // Register PNG format
	"net/url"
	"os"
	"testing"
	"time"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/models"
)

// Test storage directory and disk collection instance
var (
	storageDir        = "../teststore"
	diskCollection, _ = NewStorage(storageDir)
)

// TestDiskCollectionCreate tests creating, updating, cleaning, and removing a file in the disk collection.
func TestDiskCollectionCreate(t *testing.T) {
	var (
		testGroup   = "test"
		srcfile     = storageDir + "/bucket/file/prim.jpg"
		tags        = []string{"images", "test"}
		ctx, cancel = context.WithTimeout(context.TODO(), time.Second*10)
		file, err   = diskCollection.Create(ctx, testGroup, nil, false, url.Values{"tags": tags})
	)
	defer cancel()

	// Check for error during file creation
	if err != nil {
		t.Error(err)
		return
	}

	// Open source file for updating
	file2, err := os.Open(srcfile)
	if err != nil {
		t.Error(err)
		return
	}
	defer file2.Close()

	// Update the file with original filename and content
	if err = diskCollection.Update(ctx, file, models.OriginalFilename, file2, nil); err != nil {
		t.Error(err)
		return
	}

	// Clean the file (custom logic, e.g., remove temp data)
	if err = diskCollection.Clean(ctx, file); err != nil {
		t.Error(err)
		return
	}

	// Remove the file from disk collection
	if err = diskCollection.Remove(ctx, file); err != nil {
		t.Error(err)
		return
	}

	// Cleanup test group directory
	os.RemoveAll(storageDir + "/" + testGroup)
}

// TestDiskCollectionOpen tests opening a file and verifying its path.
func TestDiskCollectionOpen(t *testing.T) {
	file, err := diskCollection.Open(context.TODO(), npio.ObjectIDType("bucket/file"))
	if err != nil {
		t.Error(err)
		return
	}

	// Check if the file path is as expected
	if file.Path() != "file" {
		t.Error("invalid codename")
	}
}
