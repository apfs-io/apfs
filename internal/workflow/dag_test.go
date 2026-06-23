package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apfs-io/apfs/models"
)

func TestBuildDAG_Order(t *testing.T) {
	w := &models.Workflow{
		Jobs: map[string]*models.WorkflowJob{
			"probe":           {RunsOn: "any"},
			"transcode-main":  {RunsOn: "large", Needs: []string{"probe"}},
			"transcode-small": {RunsOn: "large", Needs: []string{"probe"}},
			"finalize":        {RunsOn: "any", Needs: []string{"transcode-main", "transcode-small"}},
		},
	}
	dag, err := BuildDAG(w)
	require.NoError(t, err)

	order := dag.TopologicalOrder()
	assert.Equal(t, 4, len(order))
	// probe must come before transcodes
	probeIdx := indexOf(order, "probe")
	mainIdx := indexOf(order, "transcode-main")
	smallIdx := indexOf(order, "transcode-small")
	finalIdx := indexOf(order, "finalize")
	assert.Less(t, probeIdx, mainIdx)
	assert.Less(t, probeIdx, smallIdx)
	assert.Less(t, mainIdx, finalIdx)
	assert.Less(t, smallIdx, finalIdx)
}

func TestBuildDAG_Cycle(t *testing.T) {
	w := &models.Workflow{
		Jobs: map[string]*models.WorkflowJob{
			"a": {Needs: []string{"b"}},
			"b": {Needs: []string{"a"}},
		},
	}
	_, err := BuildDAG(w)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestBuildDAG_UnknownDep(t *testing.T) {
	w := &models.Workflow{
		Jobs: map[string]*models.WorkflowJob{
			"a": {Needs: []string{"missing"}},
		},
	}
	_, err := BuildDAG(w)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown job")
}

func TestReadyJobs_WorkerAffinity(t *testing.T) {
	w := &models.Workflow{
		Jobs: map[string]*models.WorkflowJob{
			"probe":          {RunsOn: "any"},
			"transcode-main": {RunsOn: "large", Needs: []string{"probe"}},
		},
	}
	dag, err := BuildDAG(w)
	require.NoError(t, err)

	now := mustTime()
	state := &models.ProcessingState{
		ObjectID: "obj1",
		Jobs: map[string]*models.JobState{
			"probe":          {Status: models.JobStatusCompleted, FinishedAt: &now},
			"transcode-main": {Status: models.JobStatusPending},
		},
	}

	// small-only worker should NOT get transcode-main (runs-on: large)
	ready := dag.ReadyJobs(state, []string{"small"})
	assert.Empty(t, ready)

	// large worker should get it
	ready = dag.ReadyJobs(state, []string{"large"})
	assert.Equal(t, []string{"transcode-main"}, ready)

	// multi-tag worker [small, large] should also get it
	ready = dag.ReadyJobs(state, []string{"small", "large"})
	assert.Equal(t, []string{"transcode-main"}, ready)

	// "any" tag matches everything
	ready = dag.ReadyJobs(state, []string{"any"})
	assert.Equal(t, []string{"transcode-main"}, ready)

	// empty tags = untagged worker, matches everything
	ready = dag.ReadyJobs(state, nil)
	assert.Equal(t, []string{"transcode-main"}, ready)
}

func indexOf(slice []string, s string) int {
	for i, v := range slice {
		if v == s {
			return i
		}
	}
	return -1
}

func mustTime() time.Time { return time.Now() }

var _ = time.Now // ensure import is used
