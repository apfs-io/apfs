package models

type actionTester interface {
	Test(action *Action) bool
}

// ManifestTask file processing
//
//easyjson:json
type ManifestTask struct {
	ID       string     `json:"id"`
	Source   string     `json:"source,omitempty"`   // '' -> @ = original file
	Target   string     `json:"target,omitempty"`   // Name of file
	Type     ObjectType `json:"type,omitempty"`     // Target type
	Actions  []*Action  `json:"actions,omitempty"`  // Applied to source before save to target
	When     []string   `json:"when,omitempty"`     // Only if tasks are processed
	Required bool       `json:"required,omitempty"` // If not required then task can`t be skipped and processin will ends with error
}

// IsLastAction of some connverter
func (task *ManifestTask) IsLastAction(action *Action, tester actionTester) bool {
	var last *Action
	if tester == nil {
		if len(task.Actions) > 0 {
			last = task.Actions[len(task.Actions)-1]
		}
	} else {
		for _, act := range task.Actions {
			if tester.Test(act) {
				last = act
			}
		}
	}
	return last.Equal(action)
}
