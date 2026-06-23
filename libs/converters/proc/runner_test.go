package proc

import (
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apfs-io/apfs/internal/workflow"
	"github.com/apfs-io/apfs/models"
)

func testdataDir() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "testdata")
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(context.Background(), testdataDir())
	require.NoError(t, err)
	return s
}

// ─── CanRun ──────────────────────────────────────────────────────────────────

func TestStepRunnerCanRun(t *testing.T) {
	r := New(nil)
	cases := []struct {
		step *models.WorkflowStep
		want bool
	}{
		{&models.WorkflowStep{Uses: "shell"}, true},
		{&models.WorkflowStep{Uses: "procedure"}, true},
		{&models.WorkflowStep{Uses: "exec"}, true},
		{&models.WorkflowStep{Uses: "docker"}, true},
		{&models.WorkflowStep{Uses: "", Run: "echo hi"}, true},
		{&models.WorkflowStep{Uses: "", Docker: &models.WorkflowStepDocker{Image: "alpine"}}, true},
		{&models.WorkflowStep{Uses: "image"}, false},
		{&models.WorkflowStep{Uses: ""}, false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, r.CanRun(tc.step), "uses=%q run=%q", tc.step.Uses, tc.step.Run)
	}
}

// ─── Inline run: (shell) ─────────────────────────────────────────────────────

// TestInlineShellFileOutput runs an inline shell step that produces a file artifact.
func TestInlineShellFileOutput(t *testing.T) {
	r := New(nil)
	step := &models.WorkflowStep{
		Name: "to-upper",
		Uses: UsesShell,
		With: map[string]any{
			"target": "upper.txt",
			"input":  "stdin",
		},
		Run: `tr '[:lower:]' '[:upper:]'`,
	}
	in := workflow.StepInput{
		Reader: strings.NewReader("hello"),
		Meta:   &models.Meta{},
	}

	out, err := r.Run(context.Background(), step, in)
	require.NoError(t, err)
	require.NotNil(t, out.Writer, "expected file artifact")
	assert.Equal(t, "upper.txt", out.TargetPath)

	data, _ := io.ReadAll(out.Writer)
	assert.Equal(t, "HELLO", string(data))
}

// TestInlineShellMetaOutput runs an inline shell step that writes JSON to a meta field.
func TestInlineShellMetaOutput(t *testing.T) {
	r := New(nil)
	step := &models.WorkflowStep{
		Name: "count-words",
		Uses: UsesShell,
		With: map[string]any{
			"target-meta": "words",
			"input":       "stdin",
		},
		Run: `wc -w | tr -d ' \n'`,
	}
	in := workflow.StepInput{
		Reader: strings.NewReader("one two three"),
		Meta:   &models.Meta{},
	}

	out, err := r.Run(context.Background(), step, in)
	require.NoError(t, err)
	assert.Nil(t, out.Writer, "no file artifact expected")

	v, ok := out.Outputs["words"]
	require.True(t, ok)

	var n int
	require.NoError(t, json.Unmarshal(v.(json.RawMessage), &n))
	assert.Equal(t, 3, n)
}

// ─── Inline run: with string params ──────────────────────────────────────────

// TestInlineShellWithParams verifies that with: string values are injected as macros.
func TestInlineShellWithParams(t *testing.T) {
	r := New(nil)
	step := &models.WorkflowStep{
		Name: "greet",
		Uses: UsesShell,
		With: map[string]any{
			"greeting": "Hello",
			"name":     "World",
			"target":   "out.txt",
			"input":    "skip",
		},
		Run: `printf '%s, %s!\n' "{{greeting}}" "{{name}}"`,
	}
	in := workflow.StepInput{Meta: &models.Meta{}}

	out, err := r.Run(context.Background(), step, in)
	require.NoError(t, err)
	require.NotNil(t, out.Writer)

	data, _ := io.ReadAll(out.Writer)
	assert.Equal(t, "Hello, World!\n", string(data))
}

// ─── Named procedure from store ──────────────────────────────────────────────

// TestStoreProcedureFileOutput exercises calling a named .eproc procedure from the store.
func TestStoreProcedureFileOutput(t *testing.T) {
	store := newTestStore(t)
	require.NotNil(t, store.Get("echo-args"), "echo-args procedure must be loaded")

	r := New(store)
	step := &models.WorkflowStep{
		Name: "echo",
		Uses: UsesProcedure,
		With: map[string]any{
			"name": "echo-args",
			"arg1": "hello",
			"arg2": "world",
		},
	}
	in := workflow.StepInput{
		Reader: strings.NewReader("stdin-data"),
		Meta:   &models.Meta{},
	}

	out, err := r.Run(context.Background(), step, in)
	require.NoError(t, err)
	require.NotNil(t, out.Writer)

	data, _ := io.ReadAll(out.Writer)
	got := string(data)
	assert.Contains(t, got, "hello world")
	assert.Contains(t, got, "stdin-data")
}

// TestStoreProcedureMetaOutput exercises the json-meta test procedure.
func TestStoreProcedureMetaOutput(t *testing.T) {
	store := newTestStore(t)
	require.NotNil(t, store.Get("json-meta"), "json-meta procedure must be loaded")

	r := New(store)
	step := &models.WorkflowStep{
		Name: "meta",
		Uses: UsesProcedure,
		With: map[string]any{
			"name":        "json-meta",
			"label":       "test-label",
			"target-meta": "result",
		},
	}
	in := workflow.StepInput{Meta: &models.Meta{}}

	out, err := r.Run(context.Background(), step, in)
	require.NoError(t, err)

	v, ok := out.Outputs["result"]
	require.True(t, ok)

	var obj struct {
		Label string `json:"label"`
	}
	require.NoError(t, json.Unmarshal(v.(json.RawMessage), &obj))
	assert.Equal(t, "test-label", obj.Label)
}

// ─── Error cases ─────────────────────────────────────────────────────────────

// TestMissingNameError verifies that omitting with.name on a procedure step errors.
func TestMissingNameError(t *testing.T) {
	r := New(nil)
	step := &models.WorkflowStep{
		Name: "bad",
		Uses: UsesProcedure,
		With: map[string]any{}, // no "name"
	}
	_, err := r.Run(context.Background(), step, workflow.StepInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run:")
}

// TestUnknownProcedureError verifies that an unknown procedure name returns an error.
func TestUnknownProcedureError(t *testing.T) {
	r := New(nil)
	step := &models.WorkflowStep{
		Name: "bad",
		Uses: UsesProcedure,
		With: map[string]any{"name": "no-such-proc"},
	}
	_, err := r.Run(context.Background(), step, workflow.StepInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
