package v1

type updateStateI interface {
	TryBeginUpdate(key any) bool
}

// UpdateStateFunc provides wrapper of function as state interface
type UpdateStateFunc func(key any) bool

// TryBeginUpdate state update
func (f UpdateStateFunc) TryBeginUpdate(key any) bool {
	return f(key)
}
