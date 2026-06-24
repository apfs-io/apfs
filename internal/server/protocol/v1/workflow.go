// Hand-written conversion helpers between models.Workflow and the
// protobuf-generated Workflow type.
package v1

import (
	"encoding/json"

	"github.com/apfs-io/apfs/models"
)

// WorkflowFromModel converts a models.Workflow to the generated Workflow proto type.
func WorkflowFromModel(w *models.Workflow) *Workflow {
	if w == nil || w.IsEmpty() {
		return nil
	}
	pw := &Workflow{
		Version:      w.Version,
		Name:         w.Name,
		Description:  w.Description,
		ContentTypes: append([]string{}, w.ContentTypes...),
		Tags:         append([]string{}, w.Tags...),
		OriginalName: w.OriginalName,
	}
	if w.KeepOriginal != nil {
		pw.KeepOriginal = *w.KeepOriginal
	}
	if v := w.Validate; v != nil {
		pv := &WorkflowValidate{
			MaxSize:      v.MaxSize,
			MinSize:      v.MinSize,
			ContentTypes: append([]string{}, v.ContentTypes...),
		}
		for _, ch := range v.Checks {
			withJSON, _ := json.Marshal(ch.With)
			pv.Checks = append(pv.Checks, &WorkflowValidateCheck{
				Name:     ch.Name,
				Uses:     ch.Uses,
				WithJson: string(withJSON),
			})
		}
		pw.Validate = pv
	}
	for jobID, job := range w.Jobs {
		if job == nil {
			continue
		}
		pj := &WorkflowJob{
			Id:             jobID,
			RunsOn:         job.RunsOn,
			Needs:          append([]string{}, job.Needs...),
			TimeoutMinutes: int32(job.TimeoutMinutes),
			OnFailure:      job.OnFailure,
			IfExpr:         job.If,
		}
		for _, step := range job.Steps {
			if step == nil {
				continue
			}
			withJSON, _ := json.Marshal(step.With)
			pj.Steps = append(pj.Steps, &WorkflowStep{
				Name:     step.Name,
				Uses:     step.Uses,
				WithJson: string(withJSON),
			})
		}
		pw.Jobs = append(pw.Jobs, pj)
	}
	return pw
}

// WorkflowToModel converts a generated Workflow proto to models.Workflow.
func WorkflowToModel(p *Workflow) *models.Workflow {
	if p == nil {
		return nil
	}
	w := &models.Workflow{
		Version:      p.GetVersion(),
		Name:         p.GetName(),
		Description:  p.GetDescription(),
		ContentTypes: append([]string{}, p.GetContentTypes()...),
		Tags:         append([]string{}, p.GetTags()...),
		OriginalName: p.GetOriginalName(),
	}
	if p.GetKeepOriginal() {
		t := true
		w.KeepOriginal = &t
	}
	if pv := p.GetValidate(); pv != nil {
		v := &models.WorkflowValidate{
			MaxSize:      pv.GetMaxSize(),
			MinSize:      pv.GetMinSize(),
			ContentTypes: append([]string{}, pv.GetContentTypes()...),
		}
		for _, ch := range pv.GetChecks() {
			var withMap map[string]any
			_ = json.Unmarshal([]byte(ch.GetWithJson()), &withMap)
			v.Checks = append(v.Checks, &models.WorkflowValidateCheck{
				Name: ch.GetName(),
				Uses: ch.GetUses(),
				With: withMap,
			})
		}
		w.Validate = v
	}
	if len(p.GetJobs()) > 0 {
		w.Jobs = make(map[string]*models.WorkflowJob, len(p.GetJobs()))
		for _, pj := range p.GetJobs() {
			if pj == nil {
				continue
			}
			job := &models.WorkflowJob{
				RunsOn:         pj.GetRunsOn(),
				Needs:          append([]string{}, pj.GetNeeds()...),
				TimeoutMinutes: int(pj.GetTimeoutMinutes()),
				OnFailure:      pj.GetOnFailure(),
				If:             pj.GetIfExpr(),
			}
			for _, ps := range pj.GetSteps() {
				if ps == nil {
					continue
				}
				var withMap map[string]any
				_ = json.Unmarshal([]byte(ps.GetWithJson()), &withMap)
				job.Steps = append(job.Steps, &models.WorkflowStep{
					Name: ps.GetName(),
					Uses: ps.GetUses(),
					With: withMap,
				})
			}
			w.Jobs[pj.GetId()] = job
		}
	}
	return w
}
