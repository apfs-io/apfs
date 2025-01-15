//
// @project apfs 2018 - 2022
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2022
//

package models

import (
	"reflect"
	"strings"

	"github.com/demdxx/gocast/v2"
)

// Action which must be applied to source
//
//easyjson:json
type Action struct {
	Name        string         `json:"name,omitempty"`
	MustExecute bool           `json:"mustexecute,omitempty"`
	Values      map[string]any `json:"values,omitempty"`
}

// NewAction with parameters
func NewAction(name string, values ...any) *Action {
	action := &Action{Name: name}
	for i := 0; i < len(values); i += 2 {
		if action.Values == nil {
			action.Values = map[string]any{}
		}
		action.Values[gocast.Str(values[i])] = values[i+1]
	}
	return action
}

// Copy current action object
func (a *Action) Copy() *Action {
	if a == nil {
		return nil
	}
	return &Action{Name: a.Name, Values: a.Values}
}

// Value parameter as any type
func (a *Action) Value(name string, def any) any {
	val, ok := a.Values[name]
	if !ok || gocast.IsEmpty(val) {
		return def
	}
	return val
}

// ValueFloat64 parameter as float type
func (a *Action) ValueFloat64(name string, def float64) float64 {
	return gocast.Number[float64](a.Value(name, def))
}

// ValueInt64 parameter as int type
func (a *Action) ValueInt64(name string, def int64) int64 {
	return gocast.Number[int64](a.Value(name, def))
}

// ValueInt32 parameter as int type
func (a *Action) ValueInt32(name string, def int32) int32 {
	return gocast.Number[int32](a.Value(name, def))
}

// ValueBool parameter as bool type
func (a *Action) ValueBool(name string, def bool) bool {
	return gocast.Bool(a.Value(name, def))
}

// ValueString parameter as string type
func (a *Action) ValueString(name string, def string) string {
	return strings.TrimSpace(gocast.Str(a.Value(name, def)))
}

// ValueStringSlice parameter as string slice type
func (a *Action) ValueStringSlice(name string) []string {
	return gocast.AnySlice[string](a.Value(name, nil))
}

// Equal action to
func (a *Action) Equal(act *Action) bool {
	if a == nil || act == nil {
		return false
	}
	if a.Name != act.Name {
		return false
	}
	return reflect.DeepEqual(a.Values, act.Values)
}
