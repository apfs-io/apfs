package proc

import (
	"context"

	plugeproc "github.com/demdxx/plugeproc"
	decodejson "github.com/demdxx/plugeproc/decode/json"
	decodeyaml "github.com/demdxx/plugeproc/decode/yaml"
	"github.com/demdxx/plugeproc/loader/fs"
	"github.com/demdxx/plugeproc/manifest"
	"github.com/pkg/errors"
)

// Store is a registry of named procedures loaded from a directory of .eproc manifests.
// Unlike the bare plugeproc.Store it also keeps the parsed manifests so that the
// runner can inspect param declarations for named-param matching.
type Store struct {
	manifests map[string]*manifest.Manifest
}

// NewStore builds a Store by discovering all .eproc.yaml / .eproc.json manifests
// under procedureDir. Returns an empty store when procedureDir is empty.
func NewStore(ctx context.Context, procedureDir string) (*Store, error) {
	s := &Store{manifests: map[string]*manifest.Manifest{}}
	if procedureDir == "" {
		return s, nil
	}
	loader := fs.New(procedureDir,
		fs.WithDecoder(decodejson.Decoder, decodeyaml.Decoder),
	)
	manifests, err := loader.Load()
	if err != nil {
		return nil, errors.Wrap(err, "load procedures")
	}
	for _, m := range manifests {
		s.manifests[m.Name] = m
	}
	return s, nil
}

// Get returns the manifest for the named procedure, or nil if not found.
func (s *Store) Get(name string) *manifest.Manifest {
	if s == nil {
		return nil
	}
	return s.manifests[name]
}

// Exec builds a one-shot Proc from the stored manifest and calls Exec.
func (s *Store) Exec(ctx context.Context, name string, target any, params ...any) error {
	m := s.Get(name)
	if m == nil {
		return errors.Errorf("procedure %q not found", name)
	}
	p, err := plugeproc.New(m)
	if err != nil {
		return err
	}
	defer func() { _ = p.Release() }()
	return p.Exec(ctx, target, params...)
}

// Names returns the sorted list of registered procedure names.
func (s *Store) Names() []string {
	if s == nil {
		return nil
	}
	names := make([]string, 0, len(s.manifests))
	for n := range s.manifests {
		names = append(names, n)
	}
	return names
}
