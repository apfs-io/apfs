package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Workflow helpers ──────────────────────────────────────────────────────────

func TestWorkflow_ShouldKeepOriginal(t *testing.T) {
	falseVal := false
	trueVal := true

	tests := []struct {
		name string
		w    *Workflow
		want bool
	}{
		{"nil receiver", nil, true},
		{"nil field (default true)", &Workflow{}, true},
		{"explicit true", &Workflow{KeepOriginal: &trueVal}, true},
		{"explicit false", &Workflow{KeepOriginal: &falseVal}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.w.ShouldKeepOriginal())
		})
	}
}

func TestWorkflow_OriginalBaseName(t *testing.T) {
	assert.Equal(t, "original", (*Workflow)(nil).OriginalBaseName())
	assert.Equal(t, "original", (&Workflow{}).OriginalBaseName())
	assert.Equal(t, "source", (&Workflow{OriginalName: "source"}).OriginalBaseName())
}

func TestWorkflow_IsEmpty(t *testing.T) {
	assert.True(t, (*Workflow)(nil).IsEmpty())
	assert.True(t, (&Workflow{}).IsEmpty())
	assert.False(t, (&Workflow{Jobs: map[string]*WorkflowJob{"j": {}}}).IsEmpty())
	assert.False(t, (&Workflow{Validate: &WorkflowValidate{MaxSize: "1MB"}}).IsEmpty())
}

func TestWorkflow_IsValidContentType(t *testing.T) {
	tests := []struct {
		name string
		wf   *Workflow
		ct   string
		want bool
	}{
		{"nil workflow accepts all", nil, "video/mp4", true},
		{"empty list accepts all", &Workflow{}, "video/mp4", true},
		{"exact match", &Workflow{ContentTypes: []string{"image/jpeg"}}, "image/jpeg", true},
		{"exact miss", &Workflow{ContentTypes: []string{"image/jpeg"}}, "image/png", false},
		{"wildcard video/*", &Workflow{ContentTypes: []string{"video/*"}}, "video/mp4", true},
		{"wildcard miss", &Workflow{ContentTypes: []string{"video/*"}}, "image/png", false},
		{"catch-all *", &Workflow{ContentTypes: []string{"*"}}, "application/pdf", true},
		{"multi-type, second matches", &Workflow{ContentTypes: []string{"image/jpeg", "image/png"}}, "image/png", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.wf.IsValidContentType(tc.ct))
		})
	}
}

func TestWorkflow_HasTarget(t *testing.T) {
	wf := &Workflow{
		Jobs: map[string]*WorkflowJob{
			"thumbnail": {
				Steps: []*WorkflowStep{
					{Uses: "image/resize", With: map[string]any{"target": "thumb.jpg"}},
				},
			},
			"blur": {
				Steps: []*WorkflowStep{
					{Uses: "image/blur", With: map[string]any{"target": "blur.jpg"}},
				},
			},
		},
	}

	assert.True(t, wf.HasTarget("thumb.jpg"))
	assert.True(t, wf.HasTarget("blur.jpg"))
	assert.False(t, wf.HasTarget("nope.jpg"))
	assert.False(t, (*Workflow)(nil).HasTarget("thumb.jpg"))
}

func TestWorkflow_JobIDs(t *testing.T) {
	wf := &Workflow{
		Jobs: map[string]*WorkflowJob{
			"b": {}, "a": {}, "c": {},
		},
	}
	ids := wf.JobIDs()
	assert.Equal(t, []string{"a", "b", "c"}, ids, "JobIDs should be sorted")
	assert.Nil(t, (*Workflow)(nil).JobIDs())
}

// ── WorkflowValidate ──────────────────────────────────────────────────────────

func TestWorkflowValidate_SizeBytes(t *testing.T) {
	v := &WorkflowValidate{MaxSize: "2GB", MinSize: "1MB"}
	assert.Equal(t, int64(2*1024*1024*1024), v.MaxSizeBytes())
	assert.Equal(t, int64(1*1024*1024), v.MinSizeBytes())

	assert.Equal(t, int64(0), (*WorkflowValidate)(nil).MaxSizeBytes())
	assert.Equal(t, int64(0), (*WorkflowValidate)(nil).MinSizeBytes())

	assert.Equal(t, int64(500*1024), (&WorkflowValidate{MaxSize: "500KB"}).MaxSizeBytes())
	assert.Equal(t, int64(1024), (&WorkflowValidate{MaxSize: "1024"}).MaxSizeBytes())
}

// ── FailurePolicy ─────────────────────────────────────────────────────────────

func TestParseFailurePolicy(t *testing.T) {
	assert.Equal(t, FailurePolicyFail, ParseFailurePolicy(""))
	assert.Equal(t, FailurePolicyFail, ParseFailurePolicy("fail"))
	assert.Equal(t, FailurePolicyContinue, ParseFailurePolicy("continue"))

	retry3 := ParseFailurePolicy("retry:3")
	assert.Equal(t, FailurePolicyRetry, retry3&0x0F)
	assert.Equal(t, 3, retry3.MaxRetries())
	assert.Equal(t, "retry:3", retry3.String())

	// default retry count when N is missing/invalid
	retryDef := ParseFailurePolicy("retry:abc")
	assert.Equal(t, FailurePolicyRetry, retryDef&0x0F)
	assert.Equal(t, 3, retryDef.MaxRetries())

	assert.Equal(t, "fail", FailurePolicyFail.String())
	assert.Equal(t, "continue", FailurePolicyContinue.String())
}

func TestWorkflowJob_Timeout(t *testing.T) {
	assert.Zero(t, (*WorkflowJob)(nil).Timeout())
	assert.Zero(t, (&WorkflowJob{}).Timeout())
	assert.Equal(t, 5*60*1_000_000_000, int((&WorkflowJob{TimeoutMinutes: 5}).Timeout()))
}

// ── FromLegacyManifest / ToManifest ───────────────────────────────────────────

func TestFromLegacyManifest_NilInput(t *testing.T) {
	assert.Nil(t, FromLegacyManifest(nil))
}

func TestFromLegacyManifest_Basic(t *testing.T) {
	m := &Manifest{
		Version:      "1",
		ContentTypes: []string{"image/jpeg"},
		Stages: []*ManifestTaskStage{
			{
				Name: "resize",
				Tasks: []*ManifestTask{
					{
						ID:     "thumb",
						Target: "thumb.jpg",
						Actions: []*Action{
							{Name: "image/resize", Values: map[string]any{"width": 100}},
						},
					},
				},
			},
		},
	}
	wf := FromLegacyManifest(m)
	require.NotNil(t, wf)
	assert.Equal(t, "2", wf.Version)
	assert.Equal(t, []string{"image/jpeg"}, wf.ContentTypes)
	require.Contains(t, wf.Jobs, "thumb")

	job := wf.Jobs["thumb"]
	require.Len(t, job.Steps, 1)
	assert.Equal(t, "image/resize", job.Steps[0].Uses)
	assert.Equal(t, "thumb.jpg", job.Steps[0].With["target"])
}

func TestToManifest_RoundTrip(t *testing.T) {
	wf := &Workflow{
		Version:      "2",
		ContentTypes: []string{"image/jpeg"},
		Jobs: map[string]*WorkflowJob{
			"thumbnail": {
				Steps: []*WorkflowStep{
					{
						Name: "resize",
						Uses: "image/resize",
						With: map[string]any{"target": "thumb.jpg", "width": 128},
					},
				},
			},
		},
	}

	m := wf.ToManifest()
	require.NotNil(t, m)
	assert.Equal(t, "2", m.Version)
	assert.Equal(t, []string{"image/jpeg"}, m.ContentTypes)
	require.Len(t, m.Stages, 1)

	task := m.Stages[0].TaskByID("thumbnail")
	require.NotNil(t, task)
	assert.Equal(t, "thumb.jpg", task.Target)
}

func TestToManifest_NilWorkflow(t *testing.T) {
	m := (*Workflow)(nil).ToManifest()
	require.NotNil(t, m)
	assert.Empty(t, m.Stages)
}

// ── MarshalYAML ───────────────────────────────────────────────────────────────

func TestWorkflow_MarshalYAML(t *testing.T) {
	wf := &Workflow{
		Version: "2",
		Name:    "test",
		Jobs: map[string]*WorkflowJob{
			"encode": {
				Steps: []*WorkflowStep{{Uses: "ffmpeg/encode", With: map[string]any{"target": "out.mp4"}}},
			},
		},
	}
	data, err := wf.MarshalYAML()
	require.NoError(t, err)
	assert.Contains(t, string(data), "version:")
	assert.Contains(t, string(data), "encode")
}
