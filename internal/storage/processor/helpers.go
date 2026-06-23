package processor

import (
	"bytes"
	"io"
	"reflect"

	"github.com/pkg/errors"

	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

var errReaderResetPosition = errors.New(`reader can't reset position`)

func processingStatusBy(cObject storio.Object, manifest *models.Manifest, err error) models.ObjectStatus {
	if err != nil {
		return models.StatusError
	}
	updateProcessingState(cObject, manifest)
	return cObject.Status()
}

// workflowToManifest converts a Workflow to a legacy Manifest for the
// v1-style processor. Each job becomes a ManifestTask; job execution order
// is flattened by topological sort via JobIDs().
func workflowToManifest(w *models.Workflow) *models.Manifest {
	if w == nil || w.IsEmpty() {
		return &models.Manifest{}
	}
	m := &models.Manifest{
		Version:      w.Version,
		ContentTypes: w.ContentTypes,
	}
	stage := &models.ManifestTaskStage{Name: "workflow"}
	for _, jobID := range w.JobIDs() {
		job := w.Jobs[jobID]
		if job == nil {
			continue
		}
		task := &models.ManifestTask{
			ID:       jobID,
			Required: job.OnFailure == "" || string(job.OnFailure) == "fail",
		}
		for _, step := range job.Steps {
			task.Actions = append(task.Actions, &models.Action{
				Name:   step.Uses,
				Values: step.With,
			})
			if target, ok := step.With["target"].(string); ok && target != "" {
				task.Target = target
				// Source may also be declared in step params.
				if src, ok := step.With["source"].(string); ok && src != "" {
					task.Source = src
				}
				// Extract type hint if present.
				if t, ok := step.With["type"].(string); ok {
					task.Type = models.ObjectType(t)
				}
			}
		}
		if task.Source == "" {
			task.Source = "@"
		}
		stage.Tasks = append(stage.Tasks, task)
	}
	m.Stages = []*models.ManifestTaskStage{stage}
	return m.PrepareInfo()
}

func resetReader(reader io.Reader) (out io.Reader, err error) {
	switch r := reader.(type) {
	case io.ReadSeeker:
		_, err = r.Seek(0, io.SeekStart)
		out = r
	case *bytes.Buffer:
		out = bytes.NewReader(r.Bytes())
	default:
		err = errReaderResetPosition
	}
	return out, err
}

// updateProcessingState derives and applies an ObjectStatus to cObject based
// on the current ItemMeta state vs the manifest.
func updateProcessingState(cObject storio.Object, manifest *models.Manifest) {
	meta := cObject.MetaOrNew()
	if manifest == nil || manifest.TaskCount() == 0 {
		cObject.StatusUpdate(models.StatusOK)
		return
	}
	produced := 0
	for _, stage := range manifest.GetStages() {
		for _, task := range stage.Tasks {
			if task.Target == "" {
				continue
			}
			if meta.ItemByName(task.Target) != nil {
				produced++
			}
		}
	}
	targetCount := manifest.TargetCount()
	if produced == 0 && targetCount > 0 {
		cObject.StatusUpdate(models.StatusProcessing)
	} else if produced >= targetCount {
		cObject.StatusUpdate(models.StatusOK)
	} else {
		cObject.StatusUpdate(models.StatusProcessing)
	}
}

// updateProcessingStateFromWorkflow calls updateProcessingState after converting
// the workflow to a manifest (for the v1 processor).
func updateProcessingStateFromWorkflow(cObject storio.Object, wf *models.Workflow) {
	updateProcessingState(cObject, workflowToManifest(wf))
}

func isNil(v any) bool {
	return v == nil || reflect.ValueOf(v).IsNil()
}

func defStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
