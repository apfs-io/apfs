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
	assert.True(t, wf.HasTarget("thumb"))
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

// ── Prime ─────────────────────────────────────────────────────────────────────

func primeTestWorkflow() *Workflow {
	return &Workflow{
		Version: "2",
		Prime: &WorkflowJob{Steps: []*WorkflowStep{
			{Uses: "procedure", With: map[string]any{"name": "video-transcode", "crf": "22"}},
		}},
		Jobs: map[string]*WorkflowJob{
			"sd": {Steps: []*WorkflowStep{{Uses: "procedure", With: map[string]any{"target": "small.mp4"}}}},
		},
	}
}

func TestWorkflow_Prime(t *testing.T) {
	wf := primeTestWorkflow()
	assert.True(t, wf.HasPrime())
	assert.True(t, wf.HasJobs())
	assert.False(t, wf.IsEmpty())
	assert.Equal(t, []string{PrimeJobID, "sd"}, wf.JobIDs())
	assert.Same(t, wf.Prime, wf.JobByID(PrimeJobID))
	assert.Same(t, wf.Jobs["sd"], wf.JobByID("sd"))
	assert.NoError(t, wf.ValidateReserved())

	onlyPrime := &Workflow{Prime: wf.Prime}
	assert.True(t, onlyPrime.HasJobs())
	assert.Equal(t, []string{PrimeJobID}, onlyPrime.JobIDs())

	assert.False(t, (&Workflow{}).HasPrime())
	assert.Empty(t, (&Workflow{}).PrimeHash())
}

func TestWorkflow_PrimeHash(t *testing.T) {
	a, b := primeTestWorkflow(), primeTestWorkflow()
	assert.NotEmpty(t, a.PrimeHash())
	assert.Equal(t, a.PrimeHash(), b.PrimeHash())

	b.Version = "3"
	b.Jobs["sd"].Steps[0].With["crf"] = "30"
	assert.Equal(t, a.PrimeHash(), b.PrimeHash(), "only the prime block counts")

	b.Prime.Steps[0].With["crf"] = "20"
	assert.NotEqual(t, a.PrimeHash(), b.PrimeHash())
}

func TestWorkflow_ValidateReserved(t *testing.T) {
	wf := primeTestWorkflow()
	wf.Jobs[PrimeJobID] = &WorkflowJob{}
	assert.Error(t, wf.ValidateReserved(), "reserved job id")

	wf = primeTestWorkflow()
	wf.Jobs[ConditionMainRef] = &WorkflowJob{}
	assert.Error(t, wf.ValidateReserved(), "main is reserved for if:")

	wf = primeTestWorkflow()
	wf.Jobs["sd"].Needs = []string{PrimeJobID}
	assert.Error(t, wf.ValidateReserved(), "implicit needs")

	wf = primeTestWorkflow()
	wf.Prime.Needs = []string{"sd"}
	assert.Error(t, wf.ValidateReserved(), "prime needs")

	wf = primeTestWorkflow()
	wf.Prime.Steps[0].With["target"] = "main.mp4"
	assert.Error(t, wf.ValidateReserved(), "prime target")

	wf = primeTestWorkflow()
	wf.Prime.Steps[0].With["source"] = "thumb.jpg"
	assert.Error(t, wf.ValidateReserved(), "prime source")
}

func TestMeta_SkippedJobs(t *testing.T) {
	wf := primeTestWorkflow()
	meta := Meta{ManifestVersion: "2"}
	assert.Equal(t, []string{"sd"}, meta.MissingJobTargets(wf))

	meta.SetJobSkipped("sd", true)
	meta.SetJobSkipped("sd", true)
	assert.Equal(t, []string{"sd"}, meta.SkippedJobs)
	assert.True(t, meta.IsJobSkipped("sd"))
	assert.Empty(t, meta.MissingJobTargets(wf))
	assert.True(t, meta.IsConsistent(wf))

	meta.SetJobSkipped("sd", false)
	assert.Nil(t, meta.SkippedJobs)
	assert.Equal(t, []string{"sd"}, meta.MissingJobTargets(wf))

	meta.SetJobSkipped("sd", true)
	meta.CleanSubItems()
	assert.Nil(t, meta.SkippedJobs)
}

func TestMeta_PrimeIgnoredByTargets(t *testing.T) {
	wf := primeTestWorkflow()
	meta := Meta{ManifestVersion: "2", Items: []*ItemMeta{{Name: "small", NameExt: "mp4", Path: "small.mp4"}}}
	assert.Empty(t, meta.MissingJobTargets(wf))
	assert.Empty(t, meta.ExcessItems(wf))
	assert.False(t, wf.HasTarget("prim.mp4"))
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
