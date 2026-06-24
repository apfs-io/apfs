package proc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sort"
	"strings"

	plugeproc "github.com/demdxx/plugeproc"
	"github.com/demdxx/plugeproc/manifest"
	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/workflow"
	"github.com/apfs-io/apfs/models"
)

// Supported uses values handled by this runner.
const (
	UsesShell     = "shell"
	UsesProcedure = "procedure"
	UsesExec      = "exec"
	UsesDocker    = "docker"
)

// reservedWithKeys are with-map keys consumed by the runner itself and NOT
// forwarded as macro params to the script.  "name" is only reserved for
// store-lookup steps; inline run: scripts may freely use "name" as a param.
var reservedWithKeys = map[string]bool{
	"target":      true, // output file path for StepOutput.TargetPath
	"target-meta": true, // meta attribute name for JSON output
	"input":       true, // "stdin" | "file" (default) | "skip"
	"tojson":      true, // wrap raw output in a JSON string
}

// reservedStoreKeys extends reservedWithKeys for store-lookup steps.
var reservedStoreKeys = map[string]bool{
	"name":        true, // procedure name in store
	"target":      true,
	"target-meta": true,
	"input":       true,
	"tojson":      true,
}

// StepRunner is a workflow.StepRunner backed by plugeproc.
// It handles steps with uses: shell | procedure | exec | docker.
type StepRunner struct {
	store *Store
}

// New creates a StepRunner. store may be nil when no procedure directory is configured.
func New(store *Store) *StepRunner {
	if store == nil {
		store = &Store{manifests: map[string]*manifest.Manifest{}}
	}
	return &StepRunner{store: store}
}

// CanRun returns true for steps that this runner can handle.
func (r *StepRunner) CanRun(step *models.WorkflowStep) bool {
	switch step.Uses {
	case UsesShell, UsesProcedure, UsesExec, UsesDocker:
		return true
	case "":
		// A step with no uses but with a docker block or a run: block is ours.
		return step.Run != "" || step.Docker != nil
	}
	if strings.HasPrefix(step.Uses, UsesProcedure+"/") {
		return true
	}
	return false
}

// Run executes the step.
func (r *StepRunner) Run(ctx context.Context, step *models.WorkflowStep, in workflow.StepInput) (workflow.StepOutput, error) {
	m, err := r.resolveManifest(step)
	if err != nil {
		return workflow.StepOutput{}, err
	}

	p, err := plugeproc.New(m)
	if err != nil {
		return workflow.StepOutput{}, errors.Wrap(err, "build proc")
	}
	defer func() { _ = p.Release() }()

	targetMeta := withString(step.With, "target-meta", "")
	targetPath := withString(step.With, "target", "")

	// Allocate the output receiver.
	// Named procedures may use file-type output (temp file). Inline scripts
	// always write to stdout so we always capture into a buffer or ReadCloser.
	var (
		outBuf     bytes.Buffer
		outRC      io.ReadCloser
		execTarget any
	)
	if targetMeta != "" {
		execTarget = &outBuf // JSON text → meta attribute
	} else {
		// Use ReadCloser target so plugeproc can return either a temp file or stdout.
		execTarget = &outRC
	}

	// Ordered positional params for Exec.
	params, err := buildParams(m, step, in)
	if err != nil {
		return workflow.StepOutput{}, err
	}

	if err := p.Exec(ctx, execTarget, params...); err != nil {
		return workflow.StepOutput{}, errors.Wrapf(err, "exec step %q", step.Name)
	}

	so := workflow.StepOutput{Outputs: map[string]any{}}

	if targetMeta != "" {
		data := outBuf.Bytes()
		if withBool(step.With, "tojson", false) {
			data, err = json.Marshal(string(data))
			if err != nil {
				return so, errors.Wrap(err, "tojson marshal")
			}
		}
		raw := json.RawMessage(data)
		so.Outputs[targetMeta] = raw
		so.ItemMeta = &models.ItemMeta{}
		so.ItemMeta.SetExt(targetMeta, raw)
	} else if outRC != nil {
		so.Writer = outRC
		if targetPath != "" {
			so.TargetPath = targetPath
			im := &models.ItemMeta{}
			im.UpdateName(targetPath)
			so.ItemMeta = im
		}
	}

	return so, nil
}

// resolveManifest returns the plugeproc manifest for the given step, either
// by building it from the inline run: block or by looking it up from the store.
func (r *StepRunner) resolveManifest(step *models.WorkflowStep) (*manifest.Manifest, error) {
	if step.Run != "" {
		return buildInlineManifest(step), nil
	}
	if name := procedureName(step); name != "" {
		m := r.store.Get(name)
		if m == nil {
			return nil, errors.Errorf("procedure %q not found in store", name)
		}
		return m, nil
	}
	return nil, errors.Errorf("step %q: either run: or with.name must be set", step.Name)
}

// procedureName returns the store procedure name from with.name or uses: procedure/<name>.
func procedureName(step *models.WorkflowStep) string {
	if step == nil {
		return ""
	}
	if name := withString(step.With, "name", ""); name != "" {
		return name
	}
	if strings.HasPrefix(step.Uses, UsesProcedure+"/") {
		return strings.TrimPrefix(step.Uses, UsesProcedure+"/")
	}
	return ""
}

// buildInlineManifest constructs a plugeproc manifest from a step's run: block.
func buildInlineManifest(step *models.WorkflowStep) *manifest.Manifest {
	m := &manifest.Manifest{}

	// Driver.
	switch {
	case step.Docker != nil || step.Uses == UsesDocker:
		m.Driver = manifest.DriverDocker
		m.Docker = toPlugeprocDocker(step.Docker)
	default:
		m.Driver = manifest.DriverShell
	}

	// Inline script → Command + ScriptMode (same as run: | in a YAML manifest).
	m.Command = manifest.CommandArg{step.Run}
	m.ScriptMode = true

	// Input param.
	inputMode := withString(step.With, "input", "file")
	switch inputMode {
	case "stdin":
		m.Params = append(m.Params, manifest.ParamDef{Name: "inputFile", Type: "binary", Stdin: true})
	case "skip":
		// no input param
	default: // "file"
		m.Params = append(m.Params, manifest.ParamDef{Name: "inputFile", Type: "file"})
	}

	// String params from with (sorted for determinism; reserved keys skipped).
	for _, k := range sortedWithKeys(step.With, reservedWithKeys) {
		m.Params = append(m.Params, manifest.ParamDef{Name: k, Type: "string"})
	}

	// Output: inline scripts always write to stdout (binary capture).
	// The runner maps stdout to a file artifact when target is set.
	if withString(step.With, "target-meta", "") != "" {
		m.Output = manifest.OutputDef{Type: "json"}
	} else {
		m.Output = manifest.OutputDef{Type: "binary"}
	}

	m.Normalize()
	return m
}

// buildParams assembles the positional param slice for proc.Exec.
//
// The order follows the manifest's declared param list so each value lines up
// with the correct {{name}} macro.
func buildParams(m *manifest.Manifest, step *models.WorkflowStep, in workflow.StepInput) ([]any, error) {
	params := make([]any, 0, len(m.Params))
	for _, pd := range m.Params {
		switch {
		case pd.Stdin || pd.Type == "binary" || pd.Type == "file":
			// Pass the input reader; plugeproc handles stdin piping / tmp-file creation.
			if in.Reader != nil {
				params = append(params, in.Reader)
			} else {
				params = append(params, []byte(nil))
			}
		default:
			params = append(params, withString(step.With, pd.Name, ""))
		}
	}
	return params, nil
}

// toPlugeprocDocker converts a WorkflowStepDocker to a manifest.DockerConf.
func toPlugeprocDocker(d *models.WorkflowStepDocker) *manifest.DockerConf {
	if d == nil {
		return nil
	}
	return &manifest.DockerConf{
		Image:           d.Image,
		PullImage:       d.PullImage,
		RetainContainer: d.RetainContainer,
		RemoveAfterDone: d.RemoveAfterDone,
		ContainerName:   d.ContainerName,
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func withString(with map[string]any, key, def string) string {
	if with == nil {
		return def
	}
	if v, ok := with[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return def
}

func withBool(with map[string]any, key string, def bool) bool {
	if with == nil {
		return def
	}
	if v, ok := with[key]; ok {
		switch b := v.(type) {
		case bool:
			return b
		case string:
			return strings.EqualFold(b, "true") || b == "1"
		}
	}
	return def
}

// sortedWithKeys returns the non-reserved keys of with in sorted order.
func sortedWithKeys(with map[string]any, reserved map[string]bool) []string {
	keys := make([]string, 0, len(with))
	for k := range with {
		if !reserved[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}
