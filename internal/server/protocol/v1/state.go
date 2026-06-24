// Hand-written conversion helpers between models.ProcessingState and the
// protobuf-generated ProcessingState / ProcessingCounters types.
package v1

import (
	"time"

	"github.com/apfs-io/apfs/models"
)

// ProcessingStateToProto converts a models.ProcessingState to the generated
// proto type. When full is false the Jobs map is omitted (compact/counter-only
// view). Counters are always populated.
func ProcessingStateToProto(s *models.ProcessingState) *ProcessingState {
	return processingStateToProtoFull(s, true)
}

// ProcessingStateFromModel converts a models.ProcessingState to the proto
// type. Pass full=false to omit the per-job detail (compact view).
func ProcessingStateFromModel(s *models.ProcessingState, full bool) *ProcessingState {
	return processingStateToProtoFull(s, full)
}

func processingStateToProtoFull(s *models.ProcessingState, full bool) *ProcessingState {
	if s == nil {
		return nil
	}
	c := s.Counters()
	p := &ProcessingState{
		ObjectId:        s.ObjectID,
		Status:          processingStatusToProto(s.Status),
		Progress:        float32(s.Progress),
		ManifestVersion: s.ManifestVersion,
		StartedAt:       s.StartedAt.UnixMilli(),
		UpdatedAt:       s.UpdatedAt.UnixMilli(),
		Counters: &ProcessingCounters{
			Total:     int32(c.Total),
			Pending:   int32(c.Pending),
			Running:   int32(c.Running),
			Succeeded: int32(c.Succeeded),
			Failed:    int32(c.Failed),
			Skipped:   int32(c.Skipped),
		},
	}
	if s.FinishedAt != nil {
		p.FinishedAt = s.FinishedAt.UnixMilli()
	}
	if full {
		for id, js := range s.Jobs {
			job := jobStateToProto(js)
			if job != nil {
				job.Id = id
			}
			p.Jobs = append(p.Jobs, job)
		}
	}
	return p
}

func processingStatusToProto(s models.ProcessingStatus) ProcessingStatus {
	switch s {
	case models.ProcessingStatusRunning:
		return ProcessingStatus_PROCESSING_RUNNING
	case models.ProcessingStatusCompleted:
		return ProcessingStatus_PROCESSING_COMPLETED
	case models.ProcessingStatusPartial:
		return ProcessingStatus_PROCESSING_PARTIAL
	case models.ProcessingStatusFailed:
		return ProcessingStatus_PROCESSING_FAILED
	default:
		return ProcessingStatus_PROCESSING_PENDING
	}
}

func jobStateToProto(js *models.JobState) *JobState {
	if js == nil {
		return nil
	}
	p := &JobState{
		Status:   jobStatusToProto(js.Status),
		Worker:   js.Worker,
		Attempts: int32(js.Attempts),
		Error:    js.Error,
		Progress: float32(js.Progress),
	}
	if js.StartedAt != nil {
		p.StartedAt = js.StartedAt.UnixMilli()
	}
	if js.FinishedAt != nil {
		p.FinishedAt = js.FinishedAt.UnixMilli()
	}
	for _, ss := range js.Steps {
		p.Steps = append(p.Steps, &StepState{
			Name:       ss.Name,
			Status:     stepStatusToProto(ss.Status),
			DurationMs: ss.DurationMs,
			Error:      ss.Error,
		})
	}
	return p
}

func jobStatusToProto(s models.JobStatus) JobStatus {
	switch s {
	case models.JobStatusRunning:
		return JobStatus_JOB_RUNNING
	case models.JobStatusCompleted:
		return JobStatus_JOB_COMPLETED
	case models.JobStatusFailed:
		return JobStatus_JOB_FAILED
	case models.JobStatusSkipped:
		return JobStatus_JOB_SKIPPED
	default:
		return JobStatus_JOB_PENDING
	}
}

func stepStatusToProto(s models.StepStatus) StepStatus {
	switch s {
	case models.StepStatusRunning:
		return StepStatus_STEP_RUNNING
	case models.StepStatusCompleted:
		return StepStatus_STEP_COMPLETED
	case models.StepStatusFailed:
		return StepStatus_STEP_FAILED
	case models.StepStatusSkipped:
		return StepStatus_STEP_SKIPPED
	default:
		return StepStatus_STEP_PENDING
	}
}

// ProtoToProcessingState converts a generated proto ProcessingState back to the model type.
func ProtoToProcessingState(p *ProcessingState) *models.ProcessingState {
	if p == nil {
		return nil
	}
	s := &models.ProcessingState{
		ObjectID:        p.GetObjectId(),
		Status:          protoToProcessingStatus(p.GetStatus()),
		Progress:        float64(p.GetProgress()),
		ManifestVersion: p.GetManifestVersion(),
		StartedAt:       time.UnixMilli(p.GetStartedAt()),
		UpdatedAt:       time.UnixMilli(p.GetUpdatedAt()),
	}
	if p.GetFinishedAt() > 0 {
		t := time.UnixMilli(p.GetFinishedAt())
		s.FinishedAt = &t
	}
	if len(p.GetJobs()) > 0 {
		s.Jobs = make(map[string]*models.JobState, len(p.GetJobs()))
		for _, jp := range p.GetJobs() {
			if jp != nil {
				s.Jobs[jp.GetId()] = protoToJobState(jp)
			}
		}
	}
	return s
}

func protoToProcessingStatus(s ProcessingStatus) models.ProcessingStatus {
	switch s {
	case ProcessingStatus_PROCESSING_RUNNING:
		return models.ProcessingStatusRunning
	case ProcessingStatus_PROCESSING_COMPLETED:
		return models.ProcessingStatusCompleted
	case ProcessingStatus_PROCESSING_PARTIAL:
		return models.ProcessingStatusPartial
	case ProcessingStatus_PROCESSING_FAILED:
		return models.ProcessingStatusFailed
	default:
		return models.ProcessingStatusPending
	}
}

func protoToJobState(p *JobState) *models.JobState {
	if p == nil {
		return nil
	}
	js := &models.JobState{
		Status:   protoToJobStatus(p.GetStatus()),
		Worker:   p.GetWorker(),
		Attempts: int(p.GetAttempts()),
		Error:    p.GetError(),
		Progress: float64(p.GetProgress()),
	}
	if p.GetStartedAt() > 0 {
		t := time.UnixMilli(p.GetStartedAt())
		js.StartedAt = &t
	}
	if p.GetFinishedAt() > 0 {
		t := time.UnixMilli(p.GetFinishedAt())
		js.FinishedAt = &t
	}
	for _, sp := range p.GetSteps() {
		js.Steps = append(js.Steps, &models.StepState{
			Name:       sp.GetName(),
			Status:     protoToStepStatus(sp.GetStatus()),
			DurationMs: sp.GetDurationMs(),
			Error:      sp.GetError(),
		})
	}
	return js
}

func protoToJobStatus(s JobStatus) models.JobStatus {
	switch s {
	case JobStatus_JOB_RUNNING:
		return models.JobStatusRunning
	case JobStatus_JOB_COMPLETED:
		return models.JobStatusCompleted
	case JobStatus_JOB_FAILED:
		return models.JobStatusFailed
	case JobStatus_JOB_SKIPPED:
		return models.JobStatusSkipped
	default:
		return models.JobStatusPending
	}
}

func protoToStepStatus(s StepStatus) models.StepStatus {
	switch s {
	case StepStatus_STEP_RUNNING:
		return models.StepStatusRunning
	case StepStatus_STEP_COMPLETED:
		return models.StepStatusCompleted
	case StepStatus_STEP_FAILED:
		return models.StepStatusFailed
	case StepStatus_STEP_SKIPPED:
		return models.StepStatusSkipped
	default:
		return models.StepStatusPending
	}
}
