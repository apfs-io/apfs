package procedure

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/libs/converters/shell"
	"github.com/apfs-io/apfs/models"
)

// Error list...
var (
	ErrInvalidProcedureParameter = errors.New("[procedure] invalid procedure arguments")
	ErrCantCreateTmpFile         = errors.New("[procedure] invalid tmp file creation")
)

// Converter contains whole functions to convert image
type Converter struct {
	shell.Converter
	procedureDirecory string
}

// New procedure converter executor
func New(procedureDirecory string) *Converter {
	return &Converter{procedureDirecory: procedureDirecory}
}

// Name of the converter
func (ic *Converter) Name() string {
	return ActionName
}

// Test if action is suitable to perform
func (ic *Converter) Test(action *models.Action) bool {
	return action.Name == ActionName
}

// Process input file by procedure command execution
// Example:
//
//	{
//	  "name": "procedure",
//	  "values": {
//		 	 "name": "test.sh",
//	    "args": ["arg1", "arg2"],
//	    "tojson": true,
//	    "output-file": "..."
//		 }
//	}
//
// {{tmp}} – new file on the disk automaticaly removes after processing
// By default it works as STDOUT file stream
func (ic *Converter) Process(in converters.Input, out converters.Output) error {
	var (
		err                error
		action             = in.Action()
		procedureName      = action.ValueString(ParamName, "")
		procedureArguments = action.ValueStringSlice(ParamArguments)
		targetMetaField    = action.ValueString(ParamTargetMeta, "")
		toJSONString       = action.ValueString(ParamToJSONString, "")
		inputFilepath      = action.ValueString(ParamInputFile, "")
		outputFilepath     = action.ValueString(ParamOutputFile, "")
	)
	if procedureName == "" {
		return ErrInvalidProcedureParameter
	}

	// Prepare input-output file params
	for _, arg := range procedureArguments {
		switch arg {
		case "{{inputFile}}":
			inputFilepath = arg
		case "{{outputFile}}":
			outputFilepath = arg
		}
	}
	command := filepath.Join(ic.procedureDirecory, procedureName)
	if err != nil {
		return err
	}
	return ic.Execute(command, procedureArguments,
		targetMetaField, toJSONString,
		inputFilepath == "{{inputFile}}",
		outputFilepath == "{{outputFile}}",
		in, out)
}
