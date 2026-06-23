package converters

import (
	"context"
	"strings"

	"github.com/apfs-io/apfs/internal/workflow"
	"github.com/apfs-io/apfs/models"
)

// ConverterStepRunner wraps a legacy Converter so it can be used as a
// workflow.StepRunner by the v2 workflow Executor.
//
// Mapping from WorkflowStep to legacy Action:
//   - step.Uses → action.Name (the converter's ActionName)
//   - step.With  → action.Values (parameters forwarded verbatim)
//
// File output: if step.With["target"] is set, the produced io.Reader is
// stored in StepOutput with that path.
//
// Meta output: any key written to ItemMeta.Attributes by the converter is
// promoted into StepOutput.Outputs so downstream jobs can reference it via
// ${{ jobID.outputs.key }}.
type ConverterStepRunner struct {
	actionName string // e.g. "procedure", "shell"
	conv       Converter
}

// NewStepRunner wraps conv as a workflow.StepRunner that handles steps whose
// Uses field equals actionName or starts with actionName+"/".
func NewStepRunner(actionName string, conv Converter) workflow.StepRunner {
	return &ConverterStepRunner{actionName: actionName, conv: conv}
}

// CanRun returns true when step.Uses equals the wrapped action name or starts
// with "<actionName>/".
func (r *ConverterStepRunner) CanRun(step *models.WorkflowStep) bool {
	return step.Uses == r.actionName ||
		strings.HasPrefix(step.Uses, r.actionName+"/")
}

// Run executes the step by delegating to the wrapped Converter.
func (r *ConverterStepRunner) Run(_ context.Context, step *models.WorkflowStep, in workflow.StepInput) (workflow.StepOutput, error) {
	action := &models.Action{
		Name:   r.actionName,
		Values: step.With,
	}

	// Use the main item meta as the input snapshot (read-only for the converter).
	inMeta := &models.ItemMeta{}
	if in.Meta != nil {
		snap := in.Meta.Main // value copy
		inMeta = &snap
	}
	outMeta := &models.ItemMeta{}

	convIn := NewInput(in.Reader, nil, action, inMeta)
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
		if target, ok := step.With["target"].(string); ok && target != "" {
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
