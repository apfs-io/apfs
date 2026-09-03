package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Workflow is the top-level manifest describing how uploaded objects are
// validated and processed. The schema is GitHub-Actions-inspired YAML/JSON.
type Workflow struct {
	Version     string `json:"version,omitempty"     yaml:"version,omitempty"`
	Name        string `json:"name,omitempty"        yaml:"name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// ContentTypes restricts which MIME types this workflow accepts.
	// Wildcards supported: "video/*", "image/jpeg", "*".
	ContentTypes []string `json:"content_types,omitempty" yaml:"content_types,omitempty"`

	// Tags are user-defined labels propagated to every object created
	// under this workflow.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// KeepOriginal controls whether the uploaded source file is retained.
	// Defaults to true.
	KeepOriginal *bool `json:"keep_original,omitempty" yaml:"keep_original,omitempty"`

	// OriginalName is the base name (without extension) used when the
	// original file is saved. Defaults to "original".
	OriginalName string `json:"original_name,omitempty" yaml:"original_name,omitempty"`

	// Validate runs synchronously during Upload before any file is persisted.
	// A validation failure returns an error to the caller immediately.
	Validate *WorkflowValidate `json:"validate,omitempty" yaml:"validate,omitempty"`

	// Jobs is the processing DAG. Keys are job IDs; order of execution is
	// determined by the needs graph, not by map iteration order.
	Jobs map[string]*WorkflowJob `json:"jobs,omitempty" yaml:"jobs,omitempty"`
}

// ShouldKeepOriginal returns true unless keep_original is explicitly false.
func (w *Workflow) ShouldKeepOriginal() bool {
	return w == nil || w.KeepOriginal == nil || *w.KeepOriginal
}

// OriginalBaseName returns the configured original_name or the default "original".
func (w *Workflow) OriginalBaseName() string {
	if w == nil || w.OriginalName == "" {
		return "original"
	}
	return w.OriginalName
}

// IsEmpty reports whether the workflow has no jobs and no validation.
func (w *Workflow) IsEmpty() bool {
	return w == nil || (len(w.Jobs) == 0 && w.Validate == nil)
}

// IsValidContentType checks whether ct is accepted by this workflow.
// An empty ContentTypes list accepts everything.
func (w *Workflow) IsValidContentType(ct string) bool {
	if w == nil || len(w.ContentTypes) == 0 {
		return true
	}
	for _, accepted := range w.ContentTypes {
		if matchContentType(ct, accepted) {
			return true
		}
	}
	return false
}

// JobIDs returns a deterministic-order slice of job IDs (sorted by name).
func (w *Workflow) JobIDs() []string {
	if w == nil {
		return nil
	}
	ids := make([]string, 0, len(w.Jobs))
	for id := range w.Jobs {
		ids = append(ids, id)
	}
	sortStrings(ids)
	return ids
}

// WorkflowValidate defines synchronous pre-upload validation rules.
type WorkflowValidate struct {
	// MaxSize is the maximum allowed file size (e.g. "2GB", "500MB", "1024").
	MaxSize string `json:"max_size,omitempty" yaml:"max_size,omitempty"`
	// MinSize is the minimum allowed file size.
	MinSize string `json:"min_size,omitempty" yaml:"min_size,omitempty"`

	// ContentTypes is the allowed MIME types for this validation block.
	// Falls back to the workflow-level content_types when empty.
	ContentTypes []string `json:"content_types,omitempty" yaml:"content_types,omitempty"`

	// Checks are additional validation steps executed by registered converters.
	Checks []*WorkflowValidateCheck `json:"checks,omitempty" yaml:"checks,omitempty"`
}

// MaxSizeBytes parses MaxSize and returns bytes. Returns 0 if not set or invalid.
func (v *WorkflowValidate) MaxSizeBytes() int64 {
	if v == nil {
		return 0
	}
	return parseSizeString(v.MaxSize)
}

// MinSizeBytes parses MinSize and returns bytes. Returns 0 if not set or invalid.
func (v *WorkflowValidate) MinSizeBytes() int64 {
	if v == nil {
		return 0
	}
	return parseSizeString(v.MinSize)
}

// WorkflowValidateCheck is a single named validation step.
type WorkflowValidateCheck struct {
	Name string         `json:"name,omitempty" yaml:"name,omitempty"`
	Uses string         `json:"uses"           yaml:"uses"`
	With map[string]any `json:"with,omitempty" yaml:"with,omitempty"`
}

// WorkflowJob is a single node in the processing DAG.
type WorkflowJob struct {
	// RunsOn is the worker-affinity label. Accepted values: "any", "small",
	// "large", "gpu", or "label:<custom>". Defaults to "any".
	RunsOn string `json:"runs_on,omitempty" yaml:"runs-on,omitempty"`

	// Needs lists job IDs that must complete before this job can start.
	// Defines the DAG edges. An empty Needs means the job can start immediately.
	Needs []string `json:"needs,omitempty" yaml:"needs,omitempty"`

	// TimeoutMinutes is the wall-clock timeout for the whole job.
	// Zero means no timeout.
	TimeoutMinutes int `json:"timeout_minutes,omitempty" yaml:"timeout-minutes,omitempty"`

	// OnFailure controls what happens when any step in this job fails.
	// Accepted values: "fail" (default), "continue", "retry:N".
	OnFailure string `json:"on_failure,omitempty" yaml:"on-failure,omitempty"`

	// If is a Go-template-style expression evaluated against the outputs of
	// upstream jobs. When the expression evaluates to false the job is skipped.
	// Example: "${{ probe.outputs.duration < 3600 }}"
	If string `json:"if,omitempty" yaml:"if,omitempty"`

	// Steps is the ordered list of actions executed inside this job.
	Steps []*WorkflowStep `json:"steps,omitempty" yaml:"steps,omitempty"`
}

// Timeout returns the job's timeout as a time.Duration.
func (j *WorkflowJob) Timeout() time.Duration {
	if j == nil || j.TimeoutMinutes <= 0 {
		return 0
	}
	return time.Duration(j.TimeoutMinutes) * time.Minute
}

// FailurePolicy parses OnFailure and returns a FailurePolicy value.
func (j *WorkflowJob) FailurePolicy() FailurePolicy {
	if j == nil {
		return FailurePolicyFail
	}
	return ParseFailurePolicy(j.OnFailure)
}

// WorkflowStep is one action within a WorkflowJob.
//
// Execution backend is selected by the Uses field:
//
//   - "shell"     – run an inline script defined in Run (wraps bash -c)
//   - "exec"      – run an auto-discovered procedure from the procedures dir
//   - "procedure" – alias for exec; load a named .eproc manifest from the store
//   - "docker"    – run a command inside a Docker container (Docker block required)
//
// When Run is set, an ad-hoc procedure is built from the inline script and the
// With parameters. When Run is empty, the step looks up the procedure by the
// name given in With["name"] from the loaded procedure store.
type WorkflowStep struct {
	Name   string              `json:"name,omitempty"   yaml:"name,omitempty"`
	Uses   string              `json:"uses,omitempty"   yaml:"uses,omitempty"`
	Run    string              `json:"run,omitempty"    yaml:"run,omitempty"`
	With   map[string]any      `json:"with,omitempty"   yaml:"with,omitempty"`
	Docker *WorkflowStepDocker `json:"docker,omitempty" yaml:"docker,omitempty"`
}

// Target returns the step's "target" with-param as a string, or "".
func (s *WorkflowStep) Target() string {
	if s == nil || s.With == nil {
		return ""
	}
	t, _ := s.With["target"].(string)
	return t
}

// WorkflowStepDocker holds Docker-specific configuration for a step whose
// Uses field is "docker" (or when a docker: block is present on any step).
type WorkflowStepDocker struct {
	// Image is the Docker image reference (required).
	Image string `json:"image" yaml:"image"`
	// PullImage controls whether to always pull the image before running.
	PullImage bool `json:"pull_image,omitempty" yaml:"pull_image,omitempty"`
	// RetainContainer keeps the container alive across multiple Exec calls
	// (useful for stream-mode procedures like persistent ML inference).
	RetainContainer bool `json:"retain_container,omitempty" yaml:"retain_container,omitempty"`
	// RemoveAfterDone removes the container when it exits.
	RemoveAfterDone bool `json:"remove_after_done,omitempty" yaml:"remove_after_done,omitempty"`
	// ContainerName pins a specific container name (optional).
	ContainerName string `json:"container_name,omitempty" yaml:"container_name,omitempty"`
}

// FailurePolicy describes what the workflow executor does when a job fails.
type FailurePolicy int8

const (
	FailurePolicyFail     FailurePolicy = 0 // default: abort pipeline
	FailurePolicyContinue FailurePolicy = 1 // mark partial, continue
	FailurePolicyRetry    FailurePolicy = 2 // retry up to MaxRetries times
)

// MaxRetries returns the maximum number of retries encoded in the policy.
// Returns 0 for non-retry policies.
func (p FailurePolicy) MaxRetries() int {
	return int(p) >> 4
}

// ParseFailurePolicy converts an on-failure string to a FailurePolicy.
// Recognised patterns: "fail", "continue", "retry:N".
func ParseFailurePolicy(s string) FailurePolicy {
	switch {
	case s == "" || s == "fail":
		return FailurePolicyFail
	case s == "continue":
		return FailurePolicyContinue
	case strings.HasPrefix(s, "retry:"):
		n, err := strconv.Atoi(strings.TrimPrefix(s, "retry:"))
		if err != nil || n <= 0 {
			n = 3
		}
		// encode max retries in upper nibble
		return FailurePolicy(FailurePolicyRetry | FailurePolicy(n<<4))
	default:
		return FailurePolicyFail
	}
}

func (p FailurePolicy) String() string {
	switch p & 0x0F {
	case FailurePolicyContinue:
		return "continue"
	case FailurePolicyRetry:
		n := p.MaxRetries()
		if n > 0 {
			return fmt.Sprintf("retry:%d", n)
		}
		return "retry"
	default:
		return "fail"
	}
}

// MarshalJSON implements json.Marshaler so that Workflow serialises cleanly.
func (w *Workflow) MarshalJSON() ([]byte, error) {
	type alias Workflow
	return json.Marshal((*alias)(w))
}

// MarshalYAML returns the workflow serialized as YAML bytes.
func (w *Workflow) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(w)
}

// HasTarget returns true when any job step in the workflow declares the given
// filename as its "target" output parameter. Name, Path, and Fullname forms
// all match (e.g. "large" and "large.jpg" for target "large.jpg").
func (w *Workflow) HasTarget(name string) bool {
	if w == nil || name == "" {
		return false
	}
	for _, job := range w.Jobs {
		if job == nil {
			continue
		}
		for _, step := range job.Steps {
			target := step.Target()
			if target == "" {
				continue
			}
			if target == name {
				return true
			}
			item := ItemMeta{}
			item.UpdateName(target)
			src := SourceFilename(name, item.NameExt)
			if itemMatchesName(&item, name, src) {
				return true
			}
		}
	}
	return false
}

// matchContentType returns true when ct matches the pattern (supports "type/*" wildcards).
func matchContentType(ct, pattern string) bool {
	if pattern == "*" || pattern == "" || ct == pattern {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		return strings.HasPrefix(ct, strings.TrimSuffix(pattern, "*"))
	}
	return false
}

// parseSizeString converts human-readable sizes ("2GB", "500MB", "1024") to bytes.
func parseSizeString(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	units := map[string]int64{
		"KB": 1024, "MB": 1024 * 1024, "GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}
	upper := strings.ToUpper(s)
	for suffix, mul := range units {
		if strings.HasSuffix(upper, suffix) {
			num, err := strconv.ParseFloat(strings.TrimSuffix(upper, suffix), 64)
			if err != nil {
				return 0
			}
			return int64(num * float64(mul))
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// sortStrings sorts a string slice in place (avoids importing sort for small slices).
func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j] < ss[j-1]; j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}
