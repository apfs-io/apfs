package models

import "time"

// ProcessingStatus is the top-level state of an object's processing pipeline.
type ProcessingStatus string

const (
	ProcessingStatusPending   ProcessingStatus = "pending"
	ProcessingStatusRunning   ProcessingStatus = "running"
	ProcessingStatusCompleted ProcessingStatus = "completed"
	ProcessingStatusPartial   ProcessingStatus = "partial" // some jobs failed with on-failure:continue
	ProcessingStatusFailed    ProcessingStatus = "failed"
)

func (s ProcessingStatus) String() string { return string(s) }

// IsTerminal reports whether the status represents a final (non-running) state.
func (s ProcessingStatus) IsTerminal() bool {
	return s == ProcessingStatusCompleted || s == ProcessingStatusPartial || s == ProcessingStatusFailed
}

// IsSuccess reports whether processing finished without critical errors.
func (s ProcessingStatus) IsSuccess() bool {
	return s == ProcessingStatusCompleted || s == ProcessingStatusPartial
}

// JobStatus is the execution status of a single job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusSkipped   JobStatus = "skipped" // if: condition evaluated to false
)

func (s JobStatus) String() string { return string(s) }

// IsTerminal reports whether the job status is final.
func (s JobStatus) IsTerminal() bool {
	return s == JobStatusCompleted || s == JobStatusFailed || s == JobStatusSkipped
}

// StepStatus is the execution status of a single step within a job.
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

func (s StepStatus) String() string { return string(s) }

// ProcessingState tracks the full execution state of a processing pipeline
// for a single object. It lives in state.json inside the object directory
// and is also cached in the StateStore.
//
// ProcessingState is separate from Meta: Meta describes what files exist;
// ProcessingState describes how they were (or are being) created.
type ProcessingState struct {
	ObjectID        string               `json:"object_id"`
	Status          ProcessingStatus     `json:"status"`
	Progress        float64              `json:"progress"` // 0.0–1.0
	ManifestVersion string               `json:"manifest_version,omitempty"`
	Jobs            map[string]*JobState `json:"jobs,omitempty"`
	StartedAt       time.Time            `json:"started_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	FinishedAt      *time.Time           `json:"finished_at,omitempty"`
}

// NewProcessingState creates an initial pending state for objectID.
func NewProcessingState(objectID, manifestVersion string, jobIDs []string) *ProcessingState {
	now := time.Now()
	ps := &ProcessingState{
		ObjectID:        objectID,
		Status:          ProcessingStatusPending,
		ManifestVersion: manifestVersion,
		Jobs:            make(map[string]*JobState, len(jobIDs)),
		StartedAt:       now,
		UpdatedAt:       now,
	}
	for _, id := range jobIDs {
		ps.Jobs[id] = &JobState{Status: JobStatusPending}
	}
	return ps
}

// ComputeProgress recalculates Progress from the current job states.
func (ps *ProcessingState) ComputeProgress() {
	if ps == nil || len(ps.Jobs) == 0 {
		return
	}
	done := 0
	for _, j := range ps.Jobs {
		if j.Status.IsTerminal() {
			done++
		}
	}
	ps.Progress = float64(done) / float64(len(ps.Jobs))
}

// ComputeStatus derives the top-level status from individual job states
// according to the failure policies encoded in the workflow.
func (ps *ProcessingState) ComputeStatus() {
	if ps == nil {
		return
	}
	pending, running, failed, skipped, completed := 0, 0, 0, 0, 0
	for _, j := range ps.Jobs {
		switch j.Status {
		case JobStatusPending:
			pending++
		case JobStatusRunning:
			running++
		case JobStatusFailed:
			failed++
		case JobStatusSkipped:
			skipped++
		case JobStatusCompleted:
			completed++
		}
	}
	total := len(ps.Jobs)
	switch {
	case running > 0 || (pending > 0 && completed+failed+skipped > 0):
		ps.Status = ProcessingStatusRunning
	case pending == total:
		ps.Status = ProcessingStatusPending
	case failed == 0 && pending == 0 && running == 0:
		ps.Status = ProcessingStatusCompleted
	case failed > 0 && pending == 0 && running == 0:
		ps.Status = ProcessingStatusPartial
	default:
		ps.Status = ProcessingStatusRunning
	}
}

// JobState is the runtime state of one job in the processing DAG.
type JobState struct {
	Status     JobStatus      `json:"status"`
	Worker     string         `json:"worker,omitempty"`
	Attempts   int            `json:"attempts,omitempty"`
	Outputs    map[string]any `json:"outputs,omitempty"` // available to downstream jobs via ${{ jobID.outputs.key }}
	Steps      []*StepState   `json:"steps,omitempty"`
	Error      string         `json:"error,omitempty"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Progress   float64        `json:"progress,omitempty"`
}

// MarkStarted transitions the job to running state.
func (j *JobState) MarkStarted(worker string) {
	now := time.Now()
	j.Status = JobStatusRunning
	j.Worker = worker
	j.StartedAt = &now
}

// MarkCompleted transitions the job to completed state.
func (j *JobState) MarkCompleted(outputs map[string]any) {
	now := time.Now()
	j.Status = JobStatusCompleted
	j.FinishedAt = &now
	j.Progress = 1.0
	if outputs != nil {
		j.Outputs = outputs
	}
}

// MarkFailed transitions the job to failed state.
func (j *JobState) MarkFailed(err error) {
	now := time.Now()
	j.Status = JobStatusFailed
	j.FinishedAt = &now
	j.Attempts++
	if err != nil {
		j.Error = err.Error()
	}
}

// MarkSkipped transitions the job to skipped state.
func (j *JobState) MarkSkipped(reason string) {
	now := time.Now()
	j.Status = JobStatusSkipped
	j.FinishedAt = &now
	j.Error = reason
}

// ResetForRetry prepares the job for another attempt.
func (j *JobState) ResetForRetry() {
	j.Status = JobStatusPending
	j.Worker = ""
	j.StartedAt = nil
	j.FinishedAt = nil
	j.Error = ""
	j.Steps = nil
	j.Progress = 0
}

// StepState is the runtime state of one step within a job.
type StepState struct {
	Name       string     `json:"name"`
	Status     StepStatus `json:"status"`
	DurationMs int64      `json:"duration_ms,omitempty"`
	Error      string     `json:"error,omitempty"`
}
