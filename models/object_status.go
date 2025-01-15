package models

import "slices"

// ObjectStatus of the object operation
type ObjectStatus string

// Status list
const (
	StatusUndefined  ObjectStatus = "undefined"
	StatusProcessing ObjectStatus = "processing"
	StatusOK         ObjectStatus = "ok" // If processing finished
	StatusError      ObjectStatus = "error"
	StatusNotFound   ObjectStatus = "not-found"
)

var statusList = []ObjectStatus{
	StatusNotFound,
	StatusUndefined,
	StatusOK,
	StatusProcessing,
	StatusError,
}

// StatusFromString converts string to status type
func StatusFromString(st string) ObjectStatus {
	switch ObjectStatus(st) {
	case StatusProcessing, StatusOK, StatusError, StatusNotFound:
		return ObjectStatus(st)
	default:
		return StatusUndefined
	}
}

func (s ObjectStatus) String() string {
	if !s.IsEmpty() {
		return string(s)
	}
	return string(StatusUndefined)
}

// IsEmpty checks the status
func (s ObjectStatus) IsEmpty() bool {
	return s == "" || s == StatusUndefined
}

// IsProcessed status complition
func (s ObjectStatus) IsProcessed() bool {
	return s == StatusOK
}

// IsProcessing status complition
func (s ObjectStatus) IsProcessing() bool {
	return s == StatusProcessing
}

// IsError status complition
func (s ObjectStatus) IsError() bool {
	return s == StatusError
}

// Improtant returns more significant status one of two
func (s ObjectStatus) Improtant(status2 ObjectStatus) ObjectStatus {
	if slices.Index(statusList, s) > slices.Index(statusList, status2) {
		return s
	}
	return status2
}
