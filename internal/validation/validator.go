// Package validation provides synchronous pre-upload validation.
// Validation runs BEFORE any file is stored to the driver, so a failed
// validation rejects the upload at the caller boundary (400 Bad Request).
//
// The workflow's validate: block is translated into a ValidationRequest and
// passed through the Validator chain.
package validation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/apfs-io/apfs/models"
)

// ErrValidation is the base type for all validation failures.
// Callers can test with errors.As or errors.Is.
type ErrValidation struct {
	Field   string // e.g. "size", "content_type", "check:check-duration"
	Message string
}

func (e *ErrValidation) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed [%s]: %s", e.Field, e.Message)
	}
	return "validation failed: " + e.Message
}

// ValidationRequest carries the data to be validated.
type ValidationRequest struct {
	// Reader provides the file bytes. The validator may read it to detect
	// content type; it will not be consumed beyond what is necessary for
	// sniffing (at most 512 bytes).
	Reader io.Reader
	// Size is the Content-Length reported by the caller; 0 if unknown.
	Size int64
	// ContentType is the declared MIME type; may be empty.
	ContentType string
}

// Validator checks a ValidationRequest against one or more rules.
// Multiple Validators are chained via Chain.
type Validator interface {
	Validate(ctx context.Context, req *ValidationRequest) error
}

// ValidatorFunc adapts a plain function to the Validator interface.
type ValidatorFunc func(ctx context.Context, req *ValidationRequest) error

func (f ValidatorFunc) Validate(ctx context.Context, req *ValidationRequest) error {
	return f(ctx, req)
}

// Chain returns a Validator that runs each validator in order.
// The first error stops execution.
func Chain(validators ...Validator) Validator {
	return ValidatorFunc(func(ctx context.Context, req *ValidationRequest) error {
		for _, v := range validators {
			if err := v.Validate(ctx, req); err != nil {
				return err
			}
		}
		return nil
	})
}

// FromWorkflowValidate builds a Validator from a WorkflowValidate block.
// External step checks (uses: video.probe, etc.) are looked up in registry.
// If registry is nil the step checks are skipped with a warning.
func FromWorkflowValidate(v *models.WorkflowValidate, registry CheckRegistry) Validator {
	if v == nil {
		return ValidatorFunc(func(_ context.Context, _ *ValidationRequest) error { return nil })
	}

	var validators []Validator

	// Size checks
	if maxBytes := v.MaxSizeBytes(); maxBytes > 0 {
		validators = append(validators, MaxSizeValidator(maxBytes))
	}
	if minBytes := v.MinSizeBytes(); minBytes > 0 {
		validators = append(validators, MinSizeValidator(minBytes))
	}

	// Content-type checks
	if len(v.ContentTypes) > 0 {
		validators = append(validators, ContentTypeValidator(v.ContentTypes))
	}

	// Named checks (delegate to registry)
	if registry != nil {
		for _, check := range v.Checks {
			if check == nil {
				continue
			}
			if cv := registry.Find(check.Uses); cv != nil {
				check := check // capture
				validators = append(validators, ValidatorFunc(func(ctx context.Context, req *ValidationRequest) error {
					return cv.Validate(ctx, check, req)
				}))
			}
		}
	}

	return Chain(validators...)
}

// CheckRegistry maps step "uses" strings to CheckValidator implementations.
type CheckRegistry interface {
	Find(uses string) CheckValidator
}

// CheckValidator is a named validation check that can be registered and
// invoked by the validation framework.
type CheckValidator interface {
	Validate(ctx context.Context, check *models.WorkflowValidateCheck, req *ValidationRequest) error
}

// --- Built-in validators ---

// MaxSizeValidator rejects files exceeding maxBytes.
func MaxSizeValidator(maxBytes int64) Validator {
	return ValidatorFunc(func(_ context.Context, req *ValidationRequest) error {
		if req.Size > 0 && req.Size > maxBytes {
			return &ErrValidation{
				Field:   "size",
				Message: fmt.Sprintf("file size %d exceeds maximum %d bytes", req.Size, maxBytes),
			}
		}
		return nil
	})
}

// MinSizeValidator rejects files smaller than minBytes.
func MinSizeValidator(minBytes int64) Validator {
	return ValidatorFunc(func(_ context.Context, req *ValidationRequest) error {
		if req.Size > 0 && req.Size < minBytes {
			return &ErrValidation{
				Field:   "size",
				Message: fmt.Sprintf("file size %d is below minimum %d bytes", req.Size, minBytes),
			}
		}
		return nil
	})
}

// ContentTypeValidator rejects files whose content type does not match any of
// the allowed patterns. If the request ContentType is empty, up to 512 bytes
// are sniffed from the reader.
func ContentTypeValidator(allowed []string) Validator {
	return ValidatorFunc(func(_ context.Context, req *ValidationRequest) error {
		ct := req.ContentType
		if ct == "" {
			ct = detectContentType(req)
		}
		// Normalise: strip parameters (e.g. "image/jpeg; charset=utf-8")
		baseType, _, _ := mime.ParseMediaType(ct)
		if baseType == "" {
			baseType = ct
		}
		for _, pattern := range allowed {
			if matchCT(baseType, pattern) {
				return nil
			}
		}
		return &ErrValidation{
			Field:   "content_type",
			Message: fmt.Sprintf("content type %q is not allowed; accepted: %s", ct, strings.Join(allowed, ", ")),
		}
	})
}

// IsValidationError reports whether err (or any in its chain) is an ErrValidation.
func IsValidationError(err error) bool {
	var ve *ErrValidation
	return errors.As(err, &ve)
}

// detectContentType sniffs the first 512 bytes from req.Reader.
// It restores the bytes back via a MultiReader so the rest of the pipeline
// can still read the full content.
func detectContentType(req *ValidationRequest) string {
	if req.Reader == nil {
		return ""
	}
	buf := make([]byte, 512)
	n, _ := io.ReadAtLeast(req.Reader, buf, 1)
	// Prepend the read bytes back
	req.Reader = io.MultiReader(bytes.NewReader(buf[:n]), req.Reader)
	return http.DetectContentType(buf[:n])
}

// matchCT returns true when ct matches the pattern (supports "type/*" wildcards).
func matchCT(ct, pattern string) bool {
	if pattern == "*" || pattern == "" || ct == pattern {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		return strings.HasPrefix(ct, strings.TrimSuffix(pattern, "*"))
	}
	return false
}
