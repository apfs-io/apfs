package workflow

import (
	"fmt"

	"github.com/apfs-io/apfs/models"
)

// DAG represents a directed acyclic graph of workflow jobs.
// Nodes are job IDs; edges are needs relationships.
type DAG struct {
	workflow *models.Workflow
	// order is a topologically sorted list of job IDs.
	order []string
	// deps maps jobID → set of required job IDs.
	deps map[string]map[string]struct{}
	// dependents maps jobID → set of job IDs that depend on it.
	dependents map[string][]string
}

// BuildDAG constructs and validates a DAG from a Workflow.
// Returns an error if the workflow contains a cycle or references unknown jobs.
func BuildDAG(w *models.Workflow) (*DAG, error) {
	if w == nil {
		return &DAG{workflow: w, deps: map[string]map[string]struct{}{}, dependents: map[string][]string{}}, nil
	}

	d := &DAG{
		workflow:   w,
		deps:       make(map[string]map[string]struct{}, len(w.Jobs)),
		dependents: make(map[string][]string, len(w.Jobs)),
	}

	// Initialise
	for id := range w.Jobs {
		d.deps[id] = map[string]struct{}{}
		d.dependents[id] = nil
	}

	// Build edges
	for id, job := range w.Jobs {
		for _, dep := range job.Needs {
			if _, ok := w.Jobs[dep]; !ok {
				return nil, fmt.Errorf("workflow dag: job %q needs unknown job %q", id, dep)
			}
			d.deps[id][dep] = struct{}{}
			d.dependents[dep] = append(d.dependents[dep], id)
		}
	}

	// Topological sort (Kahn's algorithm)
	inDegree := make(map[string]int, len(w.Jobs))
	for id := range w.Jobs {
		inDegree[id] = len(d.deps[id])
	}

	queue := make([]string, 0, len(w.Jobs))
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sortStrings(queue)

	order := make([]string, 0, len(w.Jobs))
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)
		deps := d.dependents[node]
		sortStrings(deps)
		for _, dep := range deps {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}
	if len(order) != len(w.Jobs) {
		return nil, fmt.Errorf("workflow dag: cycle detected among jobs")
	}

	d.order = order
	return d, nil
}

// TopologicalOrder returns job IDs in topological execution order.
func (d *DAG) TopologicalOrder() []string {
	if d == nil {
		return nil
	}
	cp := make([]string, len(d.order))
	copy(cp, d.order)
	return cp
}

// ReadyJobs returns the job IDs that can be started given the current
// ProcessingState and the worker's tag set.
//
// A job is ready when:
//  1. Its status is JobStatusPending, AND
//  2. All jobs in its Needs list are in a terminal state, AND
//  3. At least one worker tag satisfies the job's runs-on field.
func (d *DAG) ReadyJobs(state *models.ProcessingState, workerTags []string) []string {
	if d == nil || state == nil || d.workflow == nil {
		return nil
	}
	var ready []string
	for _, id := range d.order {
		js, ok := state.Jobs[id]
		if !ok || js == nil || js.Status != models.JobStatusPending {
			continue
		}
		job := d.workflow.Jobs[id]
		if job == nil {
			continue
		}
		if !matchesWorker(job.RunsOn, workerTags) {
			continue
		}
		if !d.depsTerminal(id, state) {
			continue
		}
		ready = append(ready, id)
	}
	return ready
}

// depsTerminal returns true when all dependencies of jobID are terminal.
func (d *DAG) depsTerminal(jobID string, state *models.ProcessingState) bool {
	for dep := range d.deps[jobID] {
		js, ok := state.Jobs[dep]
		if !ok || js == nil || !js.Status.IsTerminal() {
			return false
		}
	}
	return true
}

// Downstream returns all job IDs that directly or transitively depend on jobID.
func (d *DAG) Downstream(jobID string) []string {
	if d == nil {
		return nil
	}
	visited := map[string]bool{}
	queue := []string{jobID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, dep := range d.dependents[cur] {
			if !visited[dep] {
				visited[dep] = true
				queue = append(queue, dep)
			}
		}
	}
	result := make([]string, 0, len(visited))
	for id := range visited {
		result = append(result, id)
	}
	sortStrings(result)
	return result
}

// matchesWorker returns true when the job's runs-on constraint is satisfied by
// at least one of the worker's tags.
//
// Matching rules:
//   - runs-on "" or "any" → always matches (no affinity constraint).
//   - empty workerTags slice → treated as ["any"] (matches everything).
//   - otherwise → at least one tag in workerTags must equal runs-on
//     (after stripping an optional "label:" prefix from runs-on).
func matchesWorker(runsOn string, workerTags []string) bool {
	if runsOn == "" || runsOn == "any" {
		return true
	}
	if len(workerTags) == 0 {
		return true
	}
	// Strip "label:" prefix to allow both "label:ffmpeg-6" and "ffmpeg-6" as tags.
	required := runsOn
	if len(required) > 6 && required[:6] == "label:" {
		required = required[6:]
	}
	for _, tag := range workerTags {
		if tag == required || tag == "any" {
			return true
		}
	}
	return false
}

// sortStrings sorts a string slice in place (insertion sort for small slices).
func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j] < ss[j-1]; j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}
