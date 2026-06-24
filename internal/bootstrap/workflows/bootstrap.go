// Package workflows seeds bucket-level workflow manifests from a directory
// layout: {workflowsDir}/{groupName}/manifest.{yaml|json}.
package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/workflow"
	"github.com/apfs-io/apfs/models"
)

// Store reads and writes bucket-level workflow manifests.
type Store interface {
	GetWorkflow(ctx context.Context, group string) (*models.Workflow, error)
	SetWorkflow(ctx context.Context, group string, w *models.Workflow) error
}

// Bootstrap loads workflow manifests from dir and applies them to storage.
// When dir is empty or missing, Bootstrap is a no-op.
func Bootstrap(ctx context.Context, store Store, dir string, reconfigure bool, logger *zap.Logger) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			if logger != nil {
				logger.Debug("workflows bootstrap: directory not found, skipping", zap.String("dir", dir))
			}
			return nil
		}
		return fmt.Errorf("workflows bootstrap: stat %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("workflows bootstrap: %q is not a directory", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("workflows bootstrap: read dir %q: %w", dir, err)
	}

	for _, entry := range entries {
		if err := bootstrapGroup(ctx, store, dir, entry, reconfigure, logger); err != nil {
			return err
		}
	}
	return nil
}

func bootstrapGroup(
	ctx context.Context,
	store Store,
	root string,
	entry os.DirEntry,
	reconfigure bool,
	logger *zap.Logger,
) error {
	if !entry.IsDir() {
		return nil
	}
	group := entry.Name()
	if group == "" || strings.HasPrefix(group, ".") {
		return nil
	}

	manifestPath, ok := findManifest(filepath.Join(root, group))
	if !ok {
		if logger != nil {
			logger.Debug("workflows bootstrap: no manifest in group dir, skipping",
				zap.String("group", group),
				zap.String("dir", filepath.Join(root, group)),
			)
		}
		return nil
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("workflows bootstrap: read %q: %w", manifestPath, err)
	}

	incoming, err := workflow.ParseWorkflow(data)
	if err != nil {
		return fmt.Errorf("workflows bootstrap: parse %q: %w", manifestPath, err)
	}
	if incoming.IsEmpty() {
		if logger != nil {
			logger.Warn("workflows bootstrap: empty manifest, skipping",
				zap.String("group", group),
				zap.String("path", manifestPath),
			)
		}
		return nil
	}

	existing, err := store.GetWorkflow(ctx, group)
	if err != nil {
		return fmt.Errorf("workflows bootstrap: read workflow for group %q: %w", group, err)
	}

	action, reason := decideAction(existing, incoming, reconfigure)
	switch action {
	case actionSkip:
		if logger != nil {
			logger.Info("workflows bootstrap: skip group",
				zap.String("group", group),
				zap.String("reason", reason),
			)
		}
		return nil
	case actionApply:
		if err := store.SetWorkflow(ctx, group, incoming); err != nil {
			return fmt.Errorf("workflows bootstrap: set workflow for group %q: %w", group, err)
		}
		if logger != nil {
			logger.Info("workflows bootstrap: applied workflow",
				zap.String("group", group),
				zap.String("path", manifestPath),
				zap.String("version", incoming.GetVersion()),
				zap.String("reason", reason),
			)
		}
		return nil
	default:
		return fmt.Errorf("workflows bootstrap: unknown action for group %q", group)
	}
}

type bootstrapAction int

const (
	actionSkip bootstrapAction = iota
	actionApply
)

func decideAction(existing, incoming *models.Workflow, reconfigure bool) (bootstrapAction, string) {
	if existing == nil || existing.IsEmpty() {
		return actionApply, "group not configured"
	}
	if !reconfigure {
		return actionSkip, "group already configured and reconfigure disabled"
	}
	switch models.CompareVersion(incoming.GetVersion(), existing.GetVersion()) {
	case 1:
		return actionApply, "incoming version is newer"
	default:
		return actionSkip, "existing version is same or newer"
	}
}

func findManifest(groupDir string) (string, bool) {
	for _, name := range []string{"manifest.yaml", "manifest.yml", "manifest.json"} {
		path := filepath.Join(groupDir, name)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}
