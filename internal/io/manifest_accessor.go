package io

import (
	"context"

	"github.com/apfs-io/apfs/models"
)

// ManifestAccessor describes manifest functions
type ManifestAccessor interface {
	// ReadManifest information method
	ReadManifest(ctx context.Context, bucket string) (*models.Manifest, error)

	// UpdateManifest information method
	UpdateManifest(ctx context.Context, bucket string, manifest *models.Manifest) error
}
