//
// @project apfs 2018 - 2020
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2020
//

package converters

import (
	"errors"
	"io"

	"github.com/apfs-io/apfs/models"
)

var (
	// ErrSkip step in case if this processing no satisfy some conditions
	ErrSkip = errors.New(`skip this step`)
)

// Input value interface
type Input interface {
	Action() *models.Action
	Task() *models.ManifestTask
	Meta() *models.ItemMeta
	ObjectReader() io.Reader
}

// Output value interface
type Output interface {
	Meta() *models.ItemMeta
	SetOutput(out io.Reader) error
	OutputWriter() (io.Writer, error)
	ObjectReader() io.Reader
	IsEqual(in Input) bool
}

// Finisher of the conversion processing
type Finisher interface {
	// Finish response of processing result
	Finish(in Input, out Output) error
}

// Converter of the file processing
type Converter interface {
	// Name of the converter driver
	Name() string

	// Test if action is suitable to perform
	Test(action *models.Action) bool

	// Process input parameters by tasks
	Process(in Input, out Output) error
}
