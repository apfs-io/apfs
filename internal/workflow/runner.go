package workflow

import (
	"context"
	"io"

	"github.com/apfs-io/apfs/models"
)

// StepRunner executes a single WorkflowStep.
// Implementations are provided by the converter registry (image, video,
// procedure, shell, etc.).
type StepRunner interface {
	// CanRun reports whether this runner handles the given step.
	CanRun(step *models.WorkflowStep) bool

	// Run executes the step. It receives the input reader, the job's outputs
	// map (for writing new outputs), and the object's meta for reading/writing
	// artifact metadata.
	Run(ctx context.Context, step *models.WorkflowStep, in StepInput) (StepOutput, error)
}

// StepInput is the read context available to a step runner.
type StepInput struct {
	// Reader provides the current file data (source artifact).
	// May be nil for meta-only operations.
	Reader io.Reader
	// ObjectID is the object being processed.
	ObjectID string
	// JobID is the current job.
	JobID string
	// Meta is the object's current metadata (read-only snapshot).
	Meta *models.Meta
	// JobOutputs contains the accumulated outputs from all jobs that completed
	// before this one (for ${{ jobID.outputs.key }} resolution).
	JobOutputs map[string]map[string]any
}

// StepOutput is produced by a step runner after execution.
type StepOutput struct {
	// Writer is the processed data to write as an artifact.
	// Nil means the step did not produce a new file artifact.
	Writer io.Reader
	// TargetPath is the relative path inside the object scope where Writer
	// should be stored. Required when Writer is non-nil.
	TargetPath string
	// ItemMeta is the metadata for the produced artifact.
	ItemMeta *models.ItemMeta
	// Outputs are key/value pairs published by this step into the job's
	// outputs map (accessible to downstream jobs via ${{ jobID.outputs.key }}).
	Outputs map[string]any
}

// RunnerRegistry is a registry of StepRunners. The executor uses it to
// dispatch individual steps to the appropriate runner.
type RunnerRegistry struct {
	runners []StepRunner
}

// NewRunnerRegistry creates an empty registry.
func NewRunnerRegistry() *RunnerRegistry {
	return &RunnerRegistry{}
}

// Register adds a runner to the registry.
func (r *RunnerRegistry) Register(runner StepRunner) {
	r.runners = append(r.runners, runner)
}

// Find returns the first runner that can handle the given step, or nil.
func (r *RunnerRegistry) Find(step *models.WorkflowStep) StepRunner {
	for _, runner := range r.runners {
		if runner.CanRun(step) {
			return runner
		}
	}
	return nil
}
