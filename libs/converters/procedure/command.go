package procedure

import "github.com/apfs-io/apfs/models"

// ActionName of command
const ActionName = "procedure"

// Command parameters
const (
	ParamName         = "name"
	ParamArguments    = "args"
	ParamTargetMeta   = "target-meta"
	ParamToJSONString = "tojson"
	ParamOutputFile   = "output-file"
	ParamInputFile    = "input-file"
)

// NewAction procedure defenition
func NewAction(procedure, outFile string, toJSON bool, args ...string) *models.Action {
	return models.NewAction(ActionName,
		ParamName, procedure,
		ParamArguments, args,
		ParamOutputFile, outFile,
		ParamToJSONString, toJSON)
}

// NewActionAsFile procedure defenition of processing as input file
func NewActionAsFile(procedure, outFile string, toJSON bool, args ...string) *models.Action {
	return models.NewAction(ActionName,
		ParamName, procedure,
		ParamArguments, args,
		ParamInputFile, "{{inputFile}}",
		ParamOutputFile, outFile,
		ParamToJSONString, toJSON)
}

// NewActionMeta extraction data
func NewActionMeta(procedure, outputTag string, toJSON bool, args ...string) *models.Action {
	return models.NewAction(ActionName,
		ParamName, procedure,
		ParamArguments, args,
		ParamTargetMeta, outputTag,
		ParamToJSONString, toJSON)
}
