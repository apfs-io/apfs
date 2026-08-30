package converters

import (
	"io"

	"github.com/apfs-io/apfs/models"
)

type input struct {
	reader io.Reader
	target string
	action *models.Action
	meta   *models.ItemMeta
}

// NewInput interface wrapper. target is the output filename (from step.With["target"]).
func NewInput(in io.Reader, target string, action *models.Action, meta *models.ItemMeta) Input {
	return &input{
		reader: in,
		target: target,
		action: action,
		meta:   meta,
	}
}

func (in *input) Action() *models.Action {
	return in.action
}

func (in *input) Target() string {
	return in.target
}

func (in *input) Meta() *models.ItemMeta {
	return in.meta
}

func (in *input) ObjectReader() io.Reader {
	if in.reader == nil {
		return nil
	}
	return in.reader
}
