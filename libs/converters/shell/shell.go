package shell

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/demdxx/gocast/v2"
	"github.com/demdxx/plugeproc"
	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/models"
)

// Error list...
var (
	ErrInvalidCommandArguments = errors.New("[shell] invalid command arguments")
	ErrCantCreateTmpFile       = errors.New("[shell] invalid tmp file creation")
	ErrShellExecute            = errors.New("[shell] execute")
	ErrShellResponse           = errors.New("[shell] command response")
	ErrInvalidJSONEncode       = errors.New("[shell] invalid JSON encode")
	ErrInvalidOutput           = errors.New("[shell] invalid output")
)

// Converter contains whole functions to convert image
type Converter struct{}

// Name of the converter
func (ic *Converter) Name() string {
	return ActionName
}

// Test if action is suitable to perform
func (ic *Converter) Test(action *models.Action) bool {
	return action.Name == ActionName
}

// Process input file by shell command execution
// Example:
//
//	{
//	  "name": "shell",
//	  "values": {
//		 	 "command": "shell exec magick - -resize \"200%\" -",
//	    "target-meta": "field-name"
//		 }
//	}
//
// {{tmp}} – new file on the disk automaticaly removes after processing
// By default it works as STDOUT file stream
func (ic *Converter) Process(in converters.Input, out converters.Output) error {
	var (
		action          = in.Action()
		command         = action.ValueString(ParamCommand, "")
		targetMetaField = action.ValueString(ParamTargetMeta, "")
		toJSONString    = action.ValueString(ParamToJSONString, "")
		outputFilepath  = action.ValueString(ParamOutputFile, "")
		inputFilepath   = action.ValueString(ParamInputFile, "")
	)
	if command == "" {
		return ErrInvalidCommandArguments
	}
	return ic.Execute(command, nil,
		targetMetaField, toJSONString,
		inputFilepath == "{{inputFile}}",
		outputFilepath == "{{outputFile}}",
		in, out)
}

// Execute the command
func (ic *Converter) Execute(
	command string, commandArgs []string,
	targetMetaField, toJSONString string,
	inputAsTempFile, outputAsTempFile bool,
	in converters.Input, out converters.Output,
) (err error) {
	var (
		output     = plugeproc.Output{Name: "outputFile", Type: "binary", AsTempFile: outputAsTempFile}
		params     = []plugeproc.Param{{Name: "inputFile", Type: "binary", IsInput: !inputAsTempFile, AsTempFile: inputAsTempFile}}
		execParams = []any{in.ObjectReader()}
		execTarget any
	)
	if targetMetaField != "" {
		output.Type = "flat"
		execTarget = &bytes.Buffer{}
	} else if outputAsTempFile {
		var v io.ReadCloser
		output.AsTempFile = true
		execTarget = &v
	} else {
		execTarget = &bytes.Buffer{}
	}

	proc, err := plugeproc.New(context.Background(), &plugeproc.Info{
		Type:      plugeproc.ProgTypeShell,
		Interface: plugeproc.IfaceDefault,
		Command:   command,
		Args:      commandArgs,
		Params:    params,
		Output:    output,
	})

	defer func() {
		if err != nil {
			switch f := out.ObjectReader().(type) {
			case nil:
			case *os.File: // Close and remove file in case of error
				if f != nil {
					filepath := f.Name()
					_ = f.Close()
					_ = os.Remove(filepath)
				}
			}
		}
	}()

	if err = proc.Exec(execTarget, execParams...); err != nil {
		return err
	}

	// Processing output value and put into meta as JSON
	if targetMetaField != "" {
		data := execTarget.(*bytes.Buffer).Bytes()
		if gocast.Bool(toJSONString) {
			data, err = json.Marshal(string(data))
			if err != nil {
				return errors.Wrap(ErrInvalidJSONEncode, err.Error())
			}
		} else if err = validateJSON(data); err != nil {
			return errors.Wrap(ErrShellResponse, err.Error())
		}
		out.Meta().SetExt(targetMetaField, json.RawMessage(data))
		return nil
	}

	// Process stdout
	switch target := execTarget.(type) {
	case *io.ReadSeeker:
		if _, err = (*target).Seek(0, io.SeekStart); err == nil {
			err = out.SetOutput(*target)
		}
	case io.ReadSeeker:
		if _, err = target.Seek(0, io.SeekStart); err == nil {
			err = out.SetOutput(target)
		}
	case *io.ReadCloser:
		err = out.SetOutput(*target)
	case *bytes.Buffer:
		err = out.SetOutput(target)
	default:
		err = ErrInvalidOutput
	}
	return err
}

// Finish response of processing result
func (ic *Converter) Finish(in converters.Input, out converters.Output) (err error) {
	switch f := out.ObjectReader().(type) {
	case nil:
	case *os.File:
		if f != nil {
			filepath := f.Name()
			err = f.Close()
			if err2 := os.Remove(filepath); err2 != nil && err == nil {
				err = err2
			}
		}
	}
	return err
}

func validateJSON(data []byte) error {
	var target any
	return json.Unmarshal(data, &target)
}
