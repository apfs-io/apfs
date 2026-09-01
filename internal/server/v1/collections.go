package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apfs-io/apfs/internal/driver/fs"
	"github.com/apfs-io/apfs/internal/driver/s3"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor/memory"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor/redis"
	"github.com/apfs-io/apfs/internal/storio"
)

// newStorage creates the new accessor collection object
func newStorage(ctx context.Context, connect string, ensureBucket *bool) (storio.StorageAccessor, error) {
	var (
		i      = strings.Index(connect, "://")
		driver = connect[:i]
	)
	switch driver {
	case "s3":
		opts := []s3.Options{s3.WithS3FromURL(connect)}
		if ensureBucket != nil {
			opts = append(opts, s3.WithEnsureBucket(*ensureBucket))
		}
		return s3.NewStorage(ctx, opts...)
	case "disk", "file", "fs":
		return fs.NewStorage(connect[i+3:])
	}
	return nil, fmt.Errorf("[storage] invalid driver: %s", driver)
}

func newKVAccessor(connect string) (kvaccessor.KVAccessor, error) {
	if connect == "memory" {
		return memory.NewKVMemory(time.Minute), nil
	}
	return redis.New(connect)
}
