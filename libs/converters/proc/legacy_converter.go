package proc

// LegacyConverter wraps the proc StepRunner as a converters.Converter so the
// legacy processor pipeline (internal/storage/processor) can continue to use
// the procedure and shell action types unchanged.
//
// Action mapping:
//   - action.Name == "procedure": look up by action.Values["name"] from store
//   - action.Name == "shell":     build ad-hoc proc from action.Values["command"]
//   - action.Name == "exec":      same as "procedure"

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	plugeproc "github.com/demdxx/plugeproc"
	"github.com/demdxx/plugeproc/manifest"
	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

const (
	// LegacyActionProcedure is the legacy action name for procedure calls.
	LegacyActionProcedure = "procedure"
	// LegacyActionShell is the legacy action name for shell commands.
	LegacyActionShell = "shell"
	// LegacyActionExec is the legacy action name (alias for procedure).
	LegacyActionExec = "exec"
)

// Legacy action parameter keys (mirror the old packages).
const (
	paramCommand      = "command"
	paramName         = "name"
	paramArguments    = "args"
	paramTargetMeta   = "target-meta"
	paramToJSONString = "tojson"
	paramOutputFile   = "output-file"
	paramInputFile    = "input-file"
)

// LegacyConverter implements converters.Converter using plugeproc as the backend.
type LegacyConverter struct {
	store *Store
}

// NewLegacyConverter creates a LegacyConverter backed by the given Store.
func NewLegacyConverter(store *Store) *LegacyConverter {
	if store == nil {
		store = &Store{manifests: map[string]*manifest.Manifest{}}
	}
	return &LegacyConverter{store: store}
}

// Name satisfies converters.Converter.
func (c *LegacyConverter) Name() string { return LegacyActionProcedure }

// Test reports whether the given action is handled by this converter.
func (c *LegacyConverter) Test(action *models.Action) bool {
	switch action.Name {
	case LegacyActionProcedure, LegacyActionShell, LegacyActionExec:
		return true
	}
	return false
}

// Process executes the action using plugeproc and writes the result to out.
func (c *LegacyConverter) Process(in converters.Input, out converters.Output) error {
	action := in.Action()

	m, err := c.resolveManifest(action)
	if err != nil {
		return err
	}

	p, err := plugeproc.New(m)
	if err != nil {
		return errors.Wrap(err, "build proc")
	}
	defer func() { _ = p.Release() }()

	targetMeta := action.ValueString(paramTargetMeta, "")
	toJSON := action.ValueString(paramToJSONString, "")

	// Build output receiver.
	var (
		outBuf     bytes.Buffer
		outRC      io.ReadCloser
		execTarget any
	)
	if targetMeta != "" {
		execTarget = &outBuf
	} else {
		execTarget = &outRC
	}

	// Positional params.
	params := buildLegacyParams(m, action, in)

	if err := p.Exec(context.Background(), execTarget, params...); err != nil {
		return err
	}

	if targetMeta != "" {
		data := outBuf.Bytes()
		if toJSON == "true" || toJSON == "1" {
			data, err = json.Marshal(string(data))
			if err != nil {
				return err
			}
		}
		out.Meta().SetExt(targetMeta, json.RawMessage(data))
		return nil
	}

	if outRC != nil {
		return out.SetOutput(outRC)
	}
	return nil
}

// Finish is a no-op (plugeproc handles cleanup internally via Release in Exec).
func (c *LegacyConverter) Finish(_ converters.Input, _ converters.Output) error { return nil }

// resolveManifest builds or looks up the plugeproc manifest from a legacy action.
func (c *LegacyConverter) resolveManifest(action *models.Action) (*manifest.Manifest, error) {
	switch action.Name {
	case LegacyActionShell:
		return c.buildShellManifest(action), nil
	default: // procedure, exec
		name := action.ValueString(paramName, "")
		if name == "" {
			return nil, errors.New("[procedure] name parameter is required")
		}
		m := c.store.Get(name)
		if m == nil {
			return nil, errors.Errorf("[procedure] %q not found in store", name)
		}
		return m, nil
	}
}

// buildShellManifest creates an ad-hoc manifest from a legacy shell action.
func (c *LegacyConverter) buildShellManifest(action *models.Action) *manifest.Manifest {
	cmd := action.ValueString(paramCommand, "")
	m := &manifest.Manifest{
		Driver:     manifest.DriverShell,
		Command:    manifest.CommandArg{cmd},
		ScriptMode: true,
	}

	inputFilepath := action.ValueString(paramInputFile, "")
	targetMeta := action.ValueString(paramTargetMeta, "")

	if inputFilepath == "{{inputFile}}" {
		m.Params = append(m.Params, manifest.ParamDef{Name: "inputFile", Type: "file"})
	} else {
		m.Params = append(m.Params, manifest.ParamDef{Name: "inputFile", Type: "binary", Stdin: true})
	}

	if targetMeta != "" {
		m.Output = manifest.OutputDef{Type: "json"}
	} else {
		m.Output = manifest.OutputDef{Type: manifest.OutputFileType, Name: "outputFile"}
	}
	m.Normalize()
	return m
}

// buildLegacyParams assembles positional Exec params from a legacy action.
func buildLegacyParams(m *manifest.Manifest, action *models.Action, in converters.Input) []any {
	params := make([]any, 0, len(m.Params))
	for _, pd := range m.Params {
		switch {
		case pd.Stdin || pd.Type == "binary" || pd.Type == "file":
			if r := in.ObjectReader(); r != nil {
				params = append(params, r)
			} else {
				params = append(params, []byte(nil))
			}
		default:
			params = append(params, action.ValueString(pd.Name, ""))
		}
	}
	return params
}
