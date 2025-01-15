package v1

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/demdxx/gocast/v2"

	"github.com/apfs-io/apfs/models"
)

// Task Action list
const (
	ActionResize        = "resize"
	ActionFit           = "fit"
	ActionFill          = "fill"
	ActionBlur          = "blur"
	ActionSharpen       = "sharpen"
	ActionGamma         = "gamma"
	ActionContrast      = "contrast"
	ActionBrightness    = "brightness"
	ActionExtractColors = "extract-colors"
)

// Error list...
var (
	ErrUnsupportedAction = errors.New("unsupported action")
)

// ActionFromModel creates new action task from model
func ActionFromModel(act *models.Action) (*Action, error) {
	action := &Action{Name: act.Name}
	if len(act.Values) == 0 {
		return action, nil
	}
	for name, val := range act.Values {
		switch v := val.(type) {
		case int, uint, int32, uint32, int64, uint64, bool:
			action.Values = append(action.Values, &Param{
				Name:      name,
				TestOneof: &Param_ValueInt{ValueInt: gocast.Number[int64](val)},
			})
		case float32, float64:
			action.Values = append(action.Values, &Param{
				Name:      name,
				TestOneof: &Param_ValueFloat{ValueFloat: gocast.Number[float64](val)},
			})
		case string:
			action.Values = append(action.Values, &Param{
				Name:      name,
				TestOneof: &Param_ValueString{ValueString: v},
			})
		case []byte:
			action.Values = append(action.Values, &Param{
				Name:      name,
				TestOneof: &Param_ValueBytes{ValueBytes: v},
			})
		case []string, []any:
			data, err := json.Marshal(gocast.AnySlice[string](v))
			if err != nil {
				return nil, err
			}
			action.Values = append(action.Values, &Param{
				Name:      name,
				TestOneof: &Param_ValueStringArray{ValueStringArray: string(data)},
			})
		default:
			return nil, fmt.Errorf("unsupported value type: %T - %v", val, val)
		}
	}
	return action, nil
}

// ToModel from protobuf object
func (m *Action) ToModel() *models.Action {
	action := &models.Action{Name: m.Name}
	for _, param := range m.Values {
		if action.Values == nil {
			action.Values = map[string]any{}
		}
		action.Values[param.GetName()] = param.Value()
	}
	return action
}

// Value golang type
func (m *Param) Value() any {
	if m.TestOneof == nil {
		return nil
	}
	switch v := m.TestOneof.(type) {
	case *Param_ValueBytes:
		return v.ValueBytes
	case *Param_ValueString:
		return v.ValueString
	case *Param_ValueInt:
		return v.ValueInt
	case *Param_ValueFloat:
		return v.ValueFloat
	case *Param_ValueStringArray:
		if v.ValueStringArray == `` {
			return nil
		}
		var arr []string
		_ = json.Unmarshal([]byte(v.ValueStringArray), &arr)
		return arr
	}
	return nil
}
