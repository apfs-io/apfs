package models

import "errors"

// EventType value
type EventType string

func (t EventType) String() string {
	return string(t)
}

// Event type list...
const (
	RefreshEventType   EventType = "refresh"
	UpdateEventType    EventType = "update"
	ProcessedEventType EventType = "processed"
	DeleteEventType    EventType = "delete"
)

// Event produced by storage
//easyjson:json
type Event struct {
	Type   EventType `json:"type"`
	Error  string    `json:"error,omitempty"`
	Object *Object   `json:"object,omitempty"`
}

// IsError object
func (e *Event) IsError() bool {
	return len(e.Error) > 0
}

// ErrorObj value
func (e *Event) ErrorObj() error {
	if len(e.Error) == 0 {
		return nil
	}
	return errors.New(e.Error)
}
