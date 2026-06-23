// Hand-written Go equivalents of state.proto messages.
// These will be replaced by generated code once protoc is re-run.
package v1

import (
	"time"

	"github.com/apfs-io/apfs/models"
)

// ProcessingStateProto is the wire representation of models.ProcessingState.
// Field names follow JSON snake_case for grpc-gateway / REST compatibility.
type ProcessingStateProto struct {
	ObjectID        string                    `json:"object_id"`
	Status          string                    `json:"status"`
	Progress        float64                   `json:"progress"`
	ManifestVersion string                    `json:"manifest_version,omitempty"`
	Jobs            map[string]*JobStateProto `json:"jobs,omitempty"`
	StartedAt       int64                     `json:"started_at"`
	UpdatedAt       int64                     `json:"updated_at"`
	FinishedAt      int64                     `json:"finished_at,omitempty"`
}

// JobStateProto is the wire representation of models.JobState.
type JobStateProto struct {
	Status     string            `json:"status"`
	Worker     string            `json:"worker,omitempty"`
	Attempts   int               `json:"attempts,omitempty"`
	Outputs    map[string]any    `json:"outputs,omitempty"`
	Steps      []*StepStateProto `json:"steps,omitempty"`
	Error      string            `json:"error,omitempty"`
	StartedAt  int64             `json:"started_at,omitempty"`
	FinishedAt int64             `json:"finished_at,omitempty"`
	Progress   float64           `json:"progress,omitempty"`
}

// StepStateProto is the wire representation of models.StepState.
type StepStateProto struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ProcessingStateToProto converts a models.ProcessingState to its wire form.
func ProcessingStateToProto(s *models.ProcessingState) *ProcessingStateProto {
	if s == nil {
		return nil
	}
	p := &ProcessingStateProto{
		ObjectID:        s.ObjectID,
		Status:          string(s.Status),
		Progress:        s.Progress,
		ManifestVersion: s.ManifestVersion,
		StartedAt:       s.StartedAt.UnixMilli(),
		UpdatedAt:       s.UpdatedAt.UnixMilli(),
	}
	if s.FinishedAt != nil {
		p.FinishedAt = s.FinishedAt.UnixMilli()
	}
	if len(s.Jobs) > 0 {
		p.Jobs = make(map[string]*JobStateProto, len(s.Jobs))
		for id, js := range s.Jobs {
			p.Jobs[id] = jobStateToProto(js)
		}
	}
	return p
}

func jobStateToProto(js *models.JobState) *JobStateProto {
	if js == nil {
		return nil
	}
	p := &JobStateProto{
		Status:   string(js.Status),
		Worker:   js.Worker,
		Attempts: js.Attempts,
		Outputs:  js.Outputs,
		Error:    js.Error,
		Progress: js.Progress,
	}
	if js.StartedAt != nil {
		p.StartedAt = js.StartedAt.UnixMilli()
	}
	if js.FinishedAt != nil {
		p.FinishedAt = js.FinishedAt.UnixMilli()
	}
	for _, ss := range js.Steps {
		p.Steps = append(p.Steps, &StepStateProto{
			Name:       ss.Name,
			Status:     string(ss.Status),
			DurationMs: ss.DurationMs,
			Error:      ss.Error,
		})
	}
	return p
}

// ProtoToProcessingState converts a wire-form state back to model.
func ProtoToProcessingState(p *ProcessingStateProto) *models.ProcessingState {
	if p == nil {
		return nil
	}
	s := &models.ProcessingState{
		ObjectID:        p.ObjectID,
		Status:          models.ProcessingStatus(p.Status),
		Progress:        p.Progress,
		ManifestVersion: p.ManifestVersion,
		StartedAt:       time.UnixMilli(p.StartedAt),
		UpdatedAt:       time.UnixMilli(p.UpdatedAt),
	}
	if p.FinishedAt > 0 {
		t := time.UnixMilli(p.FinishedAt)
		s.FinishedAt = &t
	}
	if len(p.Jobs) > 0 {
		s.Jobs = make(map[string]*models.JobState, len(p.Jobs))
		for id, jp := range p.Jobs {
			s.Jobs[id] = protoToJobState(jp)
		}
	}
	return s
}

func protoToJobState(p *JobStateProto) *models.JobState {
	if p == nil {
		return nil
	}
	js := &models.JobState{
		Status:   models.JobStatus(p.Status),
		Worker:   p.Worker,
		Attempts: p.Attempts,
		Outputs:  p.Outputs,
		Error:    p.Error,
		Progress: p.Progress,
	}
	if p.StartedAt > 0 {
		t := time.UnixMilli(p.StartedAt)
		js.StartedAt = &t
	}
	if p.FinishedAt > 0 {
		t := time.UnixMilli(p.FinishedAt)
		js.FinishedAt = &t
	}
	for _, sp := range p.Steps {
		js.Steps = append(js.Steps, &models.StepState{
			Name:       sp.Name,
			Status:     models.StepStatus(sp.Status),
			DurationMs: sp.DurationMs,
			Error:      sp.Error,
		})
	}
	return js
}
