package processor

import (
	"context"
	"fmt"

	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

func GetProcessingStatus(ctx context.Context, statusStorage kvaccessor.KVAccessor, storage Storage, cObject storio.Object) models.ObjectStatus {
	key := processingKey(cObject)
	status, _ := statusStorage.Get(ctx, key)

	if status == "" {
		wf := storage.ObjectWorkflow(ctx, cObject)
		updateProcessingStateFromWorkflow(cObject, wf)
		return cObject.Status()
	}
	return models.ObjectStatus(status)
}

func SetProcessingStatus(ctx context.Context, statusStorage kvaccessor.KVAccessor, cObject storio.Object, status models.ObjectStatus) error {
	cObject.StatusUpdate(status)
	key := processingKey(cObject)
	return statusStorage.Set(ctx, key, status.String())
}

func processingKey(obj storio.Object) string {
	return fmt.Sprintf("processing_status:%s:%s", obj.Bucket(), obj.Path())
}
