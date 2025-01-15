package v1

import (
	"fmt"
	"strings"
	"time"

	"github.com/apfs-io/apfs/internal/driver/fs"
	"github.com/apfs-io/apfs/internal/driver/s3"
	"github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor/memory"
	"github.com/apfs-io/apfs/internal/storage/kvaccessor/redis"
)

// newStorage creates the new accessor collection object
func newStorage(connect string) (io.StorageAccessor, error) {
	var (
		i      = strings.Index(connect, "://")
		driver = connect[:i]
	)
	switch driver {
	case "s3":
		return s3.NewStorage(s3.WithS3FromURL(connect))
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
