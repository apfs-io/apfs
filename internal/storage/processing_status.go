package storage

import (
	"context"
	"fmt"

	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

func getProcessingStatus(ctx context.Context, statusStorage kvaccessor.KVAccessor, store *Storage, cObject storio.Object) models.ObjectStatus {
	key := processingKey(cObject)
	status, _ := statusStorage.Get(ctx, key)

	if status == "" {
		wf := store.ObjectWorkflow(ctx, cObject)
		updateProcessingStateFromWorkflow(cObject, wf)
		return cObject.Status()
	}
	return models.ObjectStatus(status)
}

func setProcessingStatus(ctx context.Context, statusStorage kvaccessor.KVAccessor, cObject storio.Object, status models.ObjectStatus) error {
	cObject.StatusUpdate(status)
	key := processingKey(cObject)
	return statusStorage.Set(ctx, key, status.String())
}

func processingKey(obj storio.Object) string {
	return fmt.Sprintf("processing_status:%s:%s", obj.Bucket(), obj.Path())
}

func updateProcessingStateFromWorkflow(cObject storio.Object, wf *models.Workflow) {
	if wf == nil || wf.IsEmpty() {
		cObject.StatusUpdate(models.StatusOK)
		return
	}
	meta := cObject.MetaOrNew()
	if len(meta.MissingJobTargets(wf)) > 0 {
		cObject.StatusUpdate(models.StatusProcessing)
		return
	}
	cObject.StatusUpdate(models.StatusOK)
}
