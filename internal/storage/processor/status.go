package processor

import (
	"context"
	"fmt"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	"github.com/apfs-io/apfs/models"
)

func GetProcessingStatus(ctx context.Context, statusStorage kvaccessor.KVAccessor, storage Storage, cObject npio.Object) models.ObjectStatus {
	key := processingKey(cObject)
	status, _ := statusStorage.Get(ctx, key)

	if status == "" {
		manifest := storage.ObjectManifest(ctx, cObject)
		updateProcessingState(cObject, manifest)
		return cObject.Status()
	}
	return models.ObjectStatus(status)
}

func SetProcessingStatus(ctx context.Context, statusStorage kvaccessor.KVAccessor, cObject npio.Object, status models.ObjectStatus) error {
	cObject.StatusUpdate(status)
	key := processingKey(cObject)
	return statusStorage.Set(ctx, key, status.String())
}

func processingKey(obj npio.Object) string {
	return fmt.Sprintf("processing_status:%s:%s", obj.Bucket(), obj.Path())
}
