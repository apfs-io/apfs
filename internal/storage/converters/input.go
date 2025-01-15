package converters

import (
	"io"

	"github.com/apfs-io/apfs/models"
)

type input struct {
	reader io.Reader
	task   *models.ManifestTask
	action *models.Action
	meta   *models.ItemMeta
}

// NewInput interface wrapper
func NewInput(in io.Reader, task *models.ManifestTask, action *models.Action, meta *models.ItemMeta) Input {
	return &input{
		reader: in,
		task:   task,
		action: action,
		meta:   meta,
	}
}

func (in *input) Action() *models.Action {
	return in.action
}

func (in *input) Task() *models.ManifestTask {
	return in.task
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
