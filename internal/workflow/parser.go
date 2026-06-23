// Package workflow implements the APFS processing workflow engine.
// It parses v2 YAML/JSON workflow manifests, builds the execution DAG, and
// provides an executor that dispatches jobs to workers via notificationcenter.
package workflow

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/apfs-io/apfs/models"
)

// ParseWorkflow parses a YAML or JSON workflow manifest into a *models.Workflow.
// It auto-detects the format: JSON objects start with '{', everything else is
// treated as YAML.
func ParseWorkflow(data []byte) (*models.Workflow, error) {
	if len(data) == 0 {
		return &models.Workflow{}, nil
	}

	trimmed := trimLeft(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return parseWorkflowJSON(data)
	}
	return parseWorkflowYAML(data)
}

// MustParseWorkflow is like ParseWorkflow but panics on error.
func MustParseWorkflow(data []byte) *models.Workflow {
	w, err := ParseWorkflow(data)
	if err != nil {
		panic(fmt.Sprintf("workflow: parse failed: %v", err))
	}
	return w
}

func parseWorkflowYAML(data []byte) (*models.Workflow, error) {
	var w models.Workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("workflow: yaml parse: %w", err)
	}
	prepareDefaults(&w)
	return &w, nil
}

func parseWorkflowJSON(data []byte) (*models.Workflow, error) {
	var w models.Workflow
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("workflow: json parse: %w", err)
	}
	prepareDefaults(&w)
	return &w, nil
}

// prepareDefaults fills in default values for a parsed workflow.
func prepareDefaults(w *models.Workflow) {
	if w == nil {
		return
	}
	for id, job := range w.Jobs {
		if job == nil {
			w.Jobs[id] = &models.WorkflowJob{}
			job = w.Jobs[id]
		}
		if job.RunsOn == "" {
			job.RunsOn = "any"
		}
	}
}

func trimLeft(b []byte) []byte {
	for i, c := range b {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return b[i:]
		}
	}
	return nil
}
