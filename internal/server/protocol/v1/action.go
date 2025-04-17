package v1

import (
	"errors"
	"fmt"

	"github.com/demdxx/gocast/v2"
	"github.com/demdxx/xtypes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

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
		if anyVal, err := toProtoAny(val); err != nil {
			return nil, err
		} else {
			action.Values = append(action.Values, &Param{
				Name:  name,
				Value: anyVal,
			})
		}
	}
	return action, nil
}

// ToModel from protobuf object
func (m *Action) ToModel() *models.Action {
	action := &models.Action{
		Name:   m.Name,
		Values: make(map[string]any, len(m.Values)),
	}
	for _, param := range m.Values {
		action.Values[param.GetName()] = param.GoValue()
	}
	return action
}

// GoValue golang type
func (m *Param) GoValue() any {
	if m.Value == nil {
		return nil
	}
	if vl, err := fromProtoAny(m.Value); err == nil {
		return vl
	}
	return nil
}

func toProtoAny(v any) (*anypb.Any, error) {
	switch val := v.(type) {
	case proto.Message:
		return anypb.New(val)
	default:
		var (
			v2  *structpb.Value
			err error
		)
		if gocast.IsSlice(v) {
			v2, err = structpb.NewValue(gocast.AnySlice[any](v))
		} else if gocast.IsMap(v) || gocast.IsStruct(v) {
			// gocast.Map[string, any](v) is used to convert the map to a structpb.Value
			// This is necessary because structpb.Value expects a specific format for maps
			// and gocast.Map helps in ensuring the correct type conversion.
			v2, err = structpb.NewValue(gocast.Map[string, any](v))
		} else {
			v2, err = structpb.NewValue(v)
		}
		if err != nil {
			return nil, err
		}
		return anypb.New(v2)
	}
}

func fromProtoAny(a *anypb.Any) (any, error) {
	if a == nil || a.TypeUrl == "" || a.Value == nil {
		if a != nil && a.TypeUrl != "" {
			return fromProtoAnyNil(a.TypeUrl), nil
		}
		return nil, nil
	}

	// 1. structpb
	val := &structpb.Value{}
	if err := a.UnmarshalTo(val); err == nil {
		switch v := val.AsInterface().(type) {
		case []any:
			return prepareSlice(v), nil
		default:
			return v, nil
		}
	}

	// 2. wrappers
	switch a.TypeUrl {
	case "type.googleapis.com/google.protobuf.StringValue":
		msg := &wrapperspb.StringValue{}
		if err := a.UnmarshalTo(msg); err == nil {
			return msg.Value, nil
		}
	case "type.googleapis.com/google.protobuf.Int32Value":
		msg := &wrapperspb.Int32Value{}
		if err := a.UnmarshalTo(msg); err == nil {
			return msg.Value, nil
		}
	case "type.googleapis.com/google.protobuf.Int64Value":
		msg := &wrapperspb.Int64Value{}
		if err := a.UnmarshalTo(msg); err == nil {
			return msg.Value, nil
		}
	case "type.googleapis.com/google.protobuf.FloatValue":
		msg := &wrapperspb.FloatValue{}
		if err := a.UnmarshalTo(msg); err == nil {
			return msg.Value, nil
		}
	case "type.googleapis.com/google.protobuf.DoubleValue":
		msg := &wrapperspb.DoubleValue{}
		if err := a.UnmarshalTo(msg); err == nil {
			return msg.Value, nil
		}
	case "type.googleapis.com/google.protobuf.BoolValue":
		msg := &wrapperspb.BoolValue{}
		if err := a.UnmarshalTo(msg); err == nil {
			return msg.Value, nil
		}
	case "type.googleapis.com/google.protobuf.ListValue":
		msg := &structpb.ListValue{}
		if err := a.UnmarshalTo(msg); err == nil {
			return prepareSlice(msg.AsSlice()), nil
		}
	}

	return nil, fmt.Errorf("unsupported type: %s", a.TypeUrl)
}

func fromProtoAnyNil(typeUrl string) any {
	switch typeUrl {
	case "type.googleapis.com/google.protobuf.StringValue":
		return ""
	case "type.googleapis.com/google.protobuf.Int32Value":
		return int32(0)
	case "type.googleapis.com/google.protobuf.Int64Value":
		return int64(0)
	case "type.googleapis.com/google.protobuf.FloatValue":
		return float32(0)
	case "type.googleapis.com/google.protobuf.DoubleValue":
		return float64(0)
	case "type.googleapis.com/google.protobuf.BoolValue":
		return false
	case "type.googleapis.com/google.protobuf.NullValue":
		return structpb.NullValue_NULL_VALUE
	}
	return nil
}

func prepareSlice(a []any) []any {
	if len(a) == 0 {
		return nil
	}
	return xtypes.SliceApply(a, func(v any) any {
		if v == nil {
			return nil
		}
		switch v := v.(type) {
		case map[string]any:
			if len(v) == 1 {
				for k, v := range v {
					switch k {
					case "stringValue", "int32Value", "int64Value", "floatValue", "doubleValue", "boolValue":
						return v
					}
				}
			}
			return v
		default:
			return v
		}
	})
}
