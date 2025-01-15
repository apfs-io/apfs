package storerrors

import (
	"errors"
	"strings"
)

// NotFound object error type
type NotFound struct {
	objectID string
	err      error
}

// WrapNotFound error object
func WrapNotFound(objectID string, err error) *NotFound {
	return &NotFound{objectID: objectID, err: err}
}

// IsNotFound error object
func IsNotFound(err error) bool {
	switch err := err.(type) {
	case nil:
		return false
	case *NotFound:
		return err != nil
	}
	if subErr := errors.Unwrap(err); subErr != nil {
		return IsNotFound(subErr)
	}
	return strings.Contains(err.Error(), `Object NotFound:`)
}

// Error prepare error message
func (err *NotFound) Error() string {
	if err.objectID != `` && err.err != nil {
		return `Object NotFound: ` + err.objectID + `, ` + err.err.Error()
	}
	if err.objectID != `` {
		return `Object NotFound: ` + err.objectID
	}
	return `Object NotFound: ` + err.err.Error()
}

// Unwrap error from the wrapper object
func (err *NotFound) Unwrap() error { return err.err }
