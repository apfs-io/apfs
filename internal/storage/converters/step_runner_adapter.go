package converters

import (
	"context"
	"strings"

	"github.com/apfs-io/apfs/internal/workflow"
	"github.com/apfs-io/apfs/models"
)

// ConverterStepRunner wraps a Converter so it can be used as a
// workflow.StepRunner by the workflow Executor.
//
// Mapping from WorkflowStep to Action:
//   - step.Uses → action.Name (slash form "image/resize" becomes "image.resize")
//   - step.With  → action.Values (parameters forwarded verbatim)
//
// File output: if step.With["target"] is set, the produced io.Reader is
// stored in StepOutput with that path.
//
// Meta output: any key written to ItemMeta.Attributes by the converter is
// promoted into StepOutput.Outputs so downstream jobs can reference it via
// ${{ jobID.outputs.key }}.
type ConverterStepRunner struct {
	actionName string // e.g. "image", "procedure"
	conv       Converter
}

// NewStepRunner wraps conv as a workflow.StepRunner that handles steps whose
// Uses field equals actionName or starts with actionName+"/" or actionName+".".
func NewStepRunner(actionName string, conv Converter) workflow.StepRunner {
	return &ConverterStepRunner{actionName: actionName, conv: conv}
}

// CanRun returns true when step.Uses equals the wrapped action name or starts
// with "<actionName>/" or "<actionName>.".
func (r *ConverterStepRunner) CanRun(step *models.WorkflowStep) bool {
	if step == nil {
		return false
	}
	return step.Uses == r.actionName ||
		strings.HasPrefix(step.Uses, r.actionName+"/") ||
		strings.HasPrefix(step.Uses, r.actionName+".")
}

func (r *ConverterStepRunner) actionNameFor(step *models.WorkflowStep) string {
	uses := step.Uses
	if strings.HasPrefix(uses, r.actionName+"/") {
		return r.actionName + "." + strings.TrimPrefix(uses, r.actionName+"/")
	}
	if uses == r.actionName {
		return r.actionName
	}
	return uses
}

// Run executes the step by delegating to the wrapped Converter.
func (r *ConverterStepRunner) Run(_ context.Context, step *models.WorkflowStep, in workflow.StepInput) (workflow.StepOutput, error) {
	action := &models.Action{
		Name:   r.actionNameFor(step),
		Values: step.With,
	}

	// Use the main item meta as the input snapshot (read-only for the converter).
	inMeta := &models.ItemMeta{}
	if in.Meta != nil {
		snap := in.Meta.Main // value copy
		inMeta = &snap
	}
	outMeta := &models.ItemMeta{}

	target, _ := step.With["target"].(string)
	convIn := NewInput(in.Reader, target, action, inMeta)
	convOut := NewOutput(outMeta)

	if err := r.conv.Process(convIn, convOut); err != nil {
		if fin, ok := r.conv.(Finisher); ok {
			_ = fin.Finish(convIn, convOut)
		}
		return workflow.StepOutput{}, err
	}

	so := workflow.StepOutput{
		Outputs: map[string]any{},
	}

	// Wire file output.
	if reader := convOut.ObjectReader(); reader != nil {
		so.Writer = reader
		if target != "" {
			so.TargetPath = target
			outMeta.UpdateName(target)
		}
		so.ItemMeta = outMeta
	}

	// Promote Attributes written by the converter into step outputs.
	for k, v := range outMeta.Attributes {
		so.Outputs[k] = v
	}

	// Allow the converter to clean up temp files.
	if fin, ok := r.conv.(Finisher); ok {
		_ = fin.Finish(convIn, convOut)
	}

	return so, nil
}
