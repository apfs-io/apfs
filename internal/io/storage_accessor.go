package io

// StorageAccessor basic interface
type StorageAccessor interface {
	ObjectAccessor
	ObjectScanner
	ManifestAccessor
}
