package objectpath

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/apfs-io/apfs/internal/hash"
	"github.com/apfs-io/apfs/internal/utils"
)

var (
	// ErrNewObjectPathInvalid when exided tries of generation
	ErrNewObjectPathInvalid = errors.New("invalid new object-path")
)

type (
	replacerFnk func(pattern string) (string, error)
	checkerFnk  func(path string) bool

	// Option defines update generator function
	Option func(gen *BasePathgenerator)
)

// WithChecker assign check function
func WithChecker(checker checkerFnk) Option {
	return func(gen *BasePathgenerator) {
		gen.checker = checker
	}
}

// Generator interface
type Generator interface {
	Generate(prefix string) (string, error)
}

// BasePathgenerator provides basic way of generation
type BasePathgenerator struct {
	pattern   string
	replacers []replacerFnk
	checker   checkerFnk
}

// NewBasePathgenerator returns base type of objectpath generator
func NewBasePathgenerator(pattern string, options ...Option) Generator {
	gen := &BasePathgenerator{
		pattern: pattern,
	}
	for _, opt := range options {
		opt(gen)
	}
	if false ||
		strings.Contains(pattern, "{{year}}") ||
		strings.Contains(pattern, "{{month}}") ||
		strings.Contains(pattern, "{{day}}") {
		gen.replacers = append(gen.replacers, dateReplacer)
	}
	if strings.Contains(pattern, "{{md5") {
		gen.replacers = append(gen.replacers, md5Replacer)
	}
	return gen
}

// Generate new object path
func (gen *BasePathgenerator) Generate(prefix string) (path string, err error) {
	for i := 0; i < 30; i++ {
		path = gen.pattern
		for _, rep := range gen.replacers {
			if path, err = rep(path); err != nil {
				return "", err
			}
		}
		if gen.checker == nil || gen.checker(filepath.Join(prefix, path)) {
			return path, nil
		}
	}
	return path, ErrNewObjectPathInvalid
}

func dateReplacer(path string) (string, error) {
	now := time.Now()
	return strings.NewReplacer(
		"{{year}}", now.Format("2006"),
		"{{month}}", now.Format("01"),
		"{{day}}", now.Format("02"),
	).Replace(path), nil
}

func md5Replacer(path string) (string, error) {
	hash, err := hash.Md5([]byte(path + utils.RandStr(32)))
	if err != nil {
		return "", err
	}
	return strings.NewReplacer(
		"{{md5}}", hash,
		"{{md5:1}}", hash[:1],
		"{{md5:2}}", hash[1:2],
		"{{md5:3}}", hash[2:3],
	).Replace(path), nil
}
