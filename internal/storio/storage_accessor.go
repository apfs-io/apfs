package storio

// StorageAccessor is the complete interface that every storage driver must
// implement. It composes:
//
//   - ObjectAccessor      — object lifecycle (create, open, read, update, remove)
//   - ObjectFileAccessor  — hierarchical file access within an object scope
//   - ObjectScanner       — iterating over objects in a bucket
//   - WorkflowAccessor    — reading and writing bucket-level workflow manifests
//   - ObjectStateAccessor — persisting per-object processing state
type StorageAccessor interface {
	ObjectAccessor
	ObjectFileAccessor
	ObjectScanner
	WorkflowAccessor
	ObjectStateAccessor
}
