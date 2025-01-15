package v1

type updateStateI interface {
	TryBeginUpdate(key any) bool
}

// UpdateStateFnk provides wrapper of function as state interface
type UpdateStateFnk func(key any) bool

// TryBeginUpdate state update
func (f UpdateStateFnk) TryBeginUpdate(key any) bool {
	return f(key)
}
