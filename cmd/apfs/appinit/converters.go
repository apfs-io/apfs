package appinit

import (
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/cmd/apfs/appcontext"
	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/libs/converters/image"
	"github.com/apfs-io/apfs/libs/converters/procedure"
	"github.com/apfs-io/apfs/libs/converters/shell"
)

var allDefaultConvs = []string{"image", "procedure"}

// Converters from config
func Converters(conf *appcontext.StorageConfig, logger *zap.Logger) []converters.Converter {
	convs := []converters.Converter{}
	if len(conf.Converters) == 0 {
		conf.Converters = allDefaultConvs
	}
	for _, convName := range conf.Converters {
		switch convName {
		case "image":
			convs = append(convs, image.NewDefaultConverter())
		case "procedure":
			if conf.ProcedureDirectory != `` {
				convs = append(convs, procedure.New(conf.ProcedureDirectory))
			} else {
				logger.Warn(`procedure directory not defined`)
			}
		case "shell":
			convs = append(convs, &shell.Converter{})
		default:
			logger.Fatal(`undefined converter`, zap.String(`name`, convName))
		}
	}
	return convs
}
