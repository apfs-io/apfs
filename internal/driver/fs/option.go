package fs

import (
	"github.com/apfs-io/apfs/internal/io/objectpath"
)

// Option property modification
type Option func(storage *Storage)

// WithFilepathGenerator updates path generator option
func WithFilepathGenerator(gen objectpath.Generator) Option {
	return func(storage *Storage) {
		storage.pathgen = gen
	}
}

// WithFilepathPatternGenerator updates path generator option
func WithFilepathPatternGenerator(pattern string) Option {
	return func(storage *Storage) {
		if pattern != "" {
			storage.pathgen = objectpath.NewBasePathgenerator(
				pattern, objectpath.WithChecker(PathChecker))
		}
	}
}

// WithFileCache interface
func WithFileCache(fileCache, metaCache FileCacher) Option {
	return func(storage *Storage) {
		storage.fcache = fileCache
		storage.mcache = metaCache
	}
}
