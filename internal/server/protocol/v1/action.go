package v1

import (
	"encoding/json"

	"github.com/apfs-io/apfs/models"
)

// ActionFromModel converts a models.Action to its protocol representation.
// Parameter values are JSON-encoded into the ValuesJson field so that both
// the gRPC binary wire format and the REST/JSON rendering stay free of
// "@type": "type.googleapis.com/..." annotations.
func ActionFromModel(act *models.Action) (*Action, error) {
	a := &Action{Name: act.Name}
	if len(act.Values) == 0 {
		return a, nil
	}
	data, err := json.Marshal(act.Values)
	if err != nil {
		return nil, err
	}
	a.ValuesJson = string(data)
	return a, nil
}

// ToModel converts the protobuf Action back to models.Action.
func (m *Action) ToModel() *models.Action {
	act := &models.Action{Name: m.GetName()}
	if m.GetValuesJson() == "" {
		return act
	}
	if err := json.Unmarshal([]byte(m.GetValuesJson()), &act.Values); err != nil {
		// Silently return empty values on parse failure rather than panicking.
		act.Values = map[string]any{}
	}
	return act
}
