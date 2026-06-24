package workflows

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/models"
)

type memStore struct {
	byGroup map[string]*models.Workflow
}

func (s *memStore) GetWorkflow(_ context.Context, group string) (*models.Workflow, error) {
	if wf, ok := s.byGroup[group]; ok {
		return wf, nil
	}
	return &models.Workflow{}, nil
}

func (s *memStore) SetWorkflow(_ context.Context, group string, w *models.Workflow) error {
	if s.byGroup == nil {
		s.byGroup = map[string]*models.Workflow{}
	}
	s.byGroup[group] = w
	return nil
}

const sampleManifest = `version: "2"
jobs:
  resize:
    runs-on: any
    steps:
      - name: noop
        uses: shell
        run: echo ok
`

func writeGroupManifest(t *testing.T, root, group, content string) {
	t.Helper()
	dir := filepath.Join(root, group)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(content), 0o644))
}

func TestBootstrap_appliesMissingGroups(t *testing.T) {
	root := t.TempDir()
	writeGroupManifest(t, root, "images", sampleManifest)

	store := &memStore{}
	err := Bootstrap(context.Background(), store, root, false, zap.NewNop())
	require.NoError(t, err)
	require.Contains(t, store.byGroup, "images")
	assert.Equal(t, "2", store.byGroup["images"].GetVersion())
}

func TestBootstrap_skipsConfiguredWhenReconfigureDisabled(t *testing.T) {
	root := t.TempDir()
	writeGroupManifest(t, root, "images", sampleManifest)

	store := &memStore{byGroup: map[string]*models.Workflow{
		"images": {Version: "2", Jobs: map[string]*models.WorkflowJob{"x": {}}},
	}}
	err := Bootstrap(context.Background(), store, root, false, zap.NewNop())
	require.NoError(t, err)
	assert.Len(t, store.byGroup["images"].Jobs, 1)
}

func TestBootstrap_reconfiguresWhenVersionIsNewer(t *testing.T) {
	root := t.TempDir()
	writeGroupManifest(t, root, "images", `version: "3"
jobs:
  resize:
    runs-on: any
    steps:
      - name: noop
        uses: shell
        run: echo ok
`)

	store := &memStore{byGroup: map[string]*models.Workflow{
		"images": {Version: "2", Jobs: map[string]*models.WorkflowJob{"old": {}}},
	}}
	err := Bootstrap(context.Background(), store, root, true, zap.NewNop())
	require.NoError(t, err)
	assert.Equal(t, "3", store.byGroup["images"].GetVersion())
	assert.Contains(t, store.byGroup["images"].Jobs, "resize")
}

func TestBootstrap_skipsWhenVersionNotNewer(t *testing.T) {
	root := t.TempDir()
	writeGroupManifest(t, root, "images", sampleManifest)

	store := &memStore{byGroup: map[string]*models.Workflow{
		"images": {Version: "3", Jobs: map[string]*models.WorkflowJob{"keep": {}}},
	}}
	err := Bootstrap(context.Background(), store, root, true, zap.NewNop())
	require.NoError(t, err)
	assert.Contains(t, store.byGroup["images"].Jobs, "keep")
}

func TestBootstrap_noOpWhenDirMissing(t *testing.T) {
	store := &memStore{}
	err := Bootstrap(context.Background(), store, t.TempDir()+"/missing", false, zap.NewNop())
	require.NoError(t, err)
	assert.Empty(t, store.byGroup)
}

func TestDecideAction(t *testing.T) {
	incoming := &models.Workflow{Version: "2", Jobs: map[string]*models.WorkflowJob{"j": {}}}
	existing := &models.Workflow{Version: "2", Jobs: map[string]*models.WorkflowJob{"j": {}}}

	action, _ := decideAction(nil, incoming, false)
	assert.Equal(t, actionApply, action)

	action, _ = decideAction(existing, incoming, false)
	assert.Equal(t, actionSkip, action)

	action, _ = decideAction(existing, &models.Workflow{Version: "3", Jobs: incoming.Jobs}, true)
	assert.Equal(t, actionApply, action)
}
