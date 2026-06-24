package client

import (
	"time"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/models"
)

// ProcessingCounters holds aggregate job counts for a processing pipeline.
type ProcessingCounters struct {
	Total     int
	Pending   int
	Running   int
	Succeeded int
	Failed    int
	Skipped   int
}

// ProcessingState is the client-facing representation of an object's processing
// pipeline state. Jobs is nil when only a compact (counter-only) view was requested.
type ProcessingState struct {
	ObjectID        string
	Status          models.ProcessingStatus
	Progress        float64
	ManifestVersion string
	Counters        ProcessingCounters
	Jobs            map[string]*JobState // nil in compact mode (WithState)
	StartedAt       time.Time
	UpdatedAt       time.Time
	FinishedAt      *time.Time
}

// JobState is the client-facing state of one job in the processing DAG.
type JobState struct {
	Status     models.JobStatus
	Worker     string
	Attempts   int
	Error      string
	Progress   float64
	Steps      []*StepState
	StartedAt  *time.Time
	FinishedAt *time.Time
}

// StepState is the client-facing state of one step within a job.
type StepState struct {
	Name       string
	Status     models.StepStatus
	DurationMs int64
	Error      string
}

// stateFromProto converts the generated proto ProcessingState to the client type.
// When full is false the Jobs map is not populated.
func stateFromProto(p *protocol.ProcessingState, full bool) *ProcessingState {
	if p == nil {
		return nil
	}
	s := &ProcessingState{
		ObjectID:        p.GetObjectId(),
		Status:          protoProcessingStatusToModel(p.GetStatus()),
		Progress:        float64(p.GetProgress()),
		ManifestVersion: p.GetManifestVersion(),
		StartedAt:       time.UnixMilli(p.GetStartedAt()),
		UpdatedAt:       time.UnixMilli(p.GetUpdatedAt()),
	}
	if p.GetFinishedAt() > 0 {
		t := time.UnixMilli(p.GetFinishedAt())
		s.FinishedAt = &t
	}
	if c := p.GetCounters(); c != nil {
		s.Counters = ProcessingCounters{
			Total:     int(c.GetTotal()),
			Pending:   int(c.GetPending()),
			Running:   int(c.GetRunning()),
			Succeeded: int(c.GetSucceeded()),
			Failed:    int(c.GetFailed()),
			Skipped:   int(c.GetSkipped()),
		}
	}
	if full && len(p.GetJobs()) > 0 {
		s.Jobs = make(map[string]*JobState, len(p.GetJobs()))
		for _, pj := range p.GetJobs() {
			if pj == nil {
				continue
			}
			js := &JobState{
				Status:   protoJobStatusToModel(pj.GetStatus()),
				Worker:   pj.GetWorker(),
				Attempts: int(pj.GetAttempts()),
				Error:    pj.GetError(),
				Progress: float64(pj.GetProgress()),
			}
			if pj.GetStartedAt() > 0 {
				t := time.UnixMilli(pj.GetStartedAt())
				js.StartedAt = &t
			}
			if pj.GetFinishedAt() > 0 {
				t := time.UnixMilli(pj.GetFinishedAt())
				js.FinishedAt = &t
			}
			for _, sp := range pj.GetSteps() {
				js.Steps = append(js.Steps, &StepState{
					Name:       sp.GetName(),
					Status:     protoStepStatusToModel(sp.GetStatus()),
					DurationMs: sp.GetDurationMs(),
					Error:      sp.GetError(),
				})
			}
			s.Jobs[pj.GetId()] = js
		}
	}
	return s
}

func protoProcessingStatusToModel(s protocol.ProcessingStatus) models.ProcessingStatus {
	switch s {
	case protocol.ProcessingStatus_PROCESSING_RUNNING:
		return models.ProcessingStatusRunning
	case protocol.ProcessingStatus_PROCESSING_COMPLETED:
		return models.ProcessingStatusCompleted
	case protocol.ProcessingStatus_PROCESSING_PARTIAL:
		return models.ProcessingStatusPartial
	case protocol.ProcessingStatus_PROCESSING_FAILED:
		return models.ProcessingStatusFailed
	default:
		return models.ProcessingStatusPending
	}
}

func protoJobStatusToModel(s protocol.JobStatus) models.JobStatus {
	switch s {
	case protocol.JobStatus_JOB_RUNNING:
		return models.JobStatusRunning
	case protocol.JobStatus_JOB_COMPLETED:
		return models.JobStatusCompleted
	case protocol.JobStatus_JOB_FAILED:
		return models.JobStatusFailed
	case protocol.JobStatus_JOB_SKIPPED:
		return models.JobStatusSkipped
	default:
		return models.JobStatusPending
	}
}

func protoStepStatusToModel(s protocol.StepStatus) models.StepStatus {
	switch s {
	case protocol.StepStatus_STEP_RUNNING:
		return models.StepStatusRunning
	case protocol.StepStatus_STEP_COMPLETED:
		return models.StepStatusCompleted
	case protocol.StepStatus_STEP_FAILED:
		return models.StepStatusFailed
	case protocol.StepStatus_STEP_SKIPPED:
		return models.StepStatusSkipped
	default:
		return models.StepStatusPending
	}
}
