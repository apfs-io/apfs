package appinit

import (
	"context"

	"go.uber.org/zap"

	"github.com/apfs-io/apfs/cmd/apfs/appcontext"
	"github.com/apfs-io/apfs/internal/storage/converters"
	"github.com/apfs-io/apfs/internal/workflow"
	"github.com/apfs-io/apfs/libs/converters/image"
	"github.com/apfs-io/apfs/libs/converters/proc"
)

var allDefaultConvs = []string{"image", "procedure"}

// ProcStore loads the procedure store from config. Returns nil (and logs a
// warning) when no procedure directory is configured.
func ProcStore(ctx context.Context, conf *appcontext.StorageConfig, logger *zap.Logger) *proc.Store {
	if conf.ProcedureDirectory == "" {
		logger.Warn("procedure directory not defined, procedure/shell steps disabled")
		return nil
	}
	store, err := proc.NewStore(ctx, conf.ProcedureDirectory)
	if err != nil {
		logger.Error("failed to load procedure store", zap.Error(err))
		return nil
	}
	return store
}

// Converters builds the list of legacy converters.Converter instances used by
// the internal processor pipeline.
func Converters(ctx context.Context, conf *appcontext.StorageConfig, logger *zap.Logger) []converters.Converter {
	convs := []converters.Converter{}
	if len(conf.Converters) == 0 {
		conf.Converters = allDefaultConvs
	}
	store := ProcStore(ctx, conf, logger)
	for _, convName := range conf.Converters {
		switch convName {
		case "image":
			convs = append(convs, image.NewDefaultConverter())
		case "procedure", "shell", "exec":
			convs = append(convs, proc.NewLegacyConverter(store))
		default:
			logger.Fatal("undefined converter", zap.String("name", convName))
		}
	}
	return convs
}

// StepRunners builds a workflow.RunnerRegistry populated from the same
// converter config as Converters(). Each enabled converter is wrapped as a
// workflow.StepRunner so the v2 Executor can dispatch YAML workflow steps.
func StepRunners(ctx context.Context, conf *appcontext.StorageConfig, logger *zap.Logger) *workflow.RunnerRegistry {
	reg := workflow.NewRunnerRegistry()
	if len(conf.Converters) == 0 {
		conf.Converters = allDefaultConvs
	}
	store := ProcStore(ctx, conf, logger)
	for _, convName := range conf.Converters {
		switch convName {
		case "image":
			reg.Register(image.NewDefaultConverter().StepRunner())
		case "procedure", "shell", "exec":
			reg.Register(proc.New(store))
		default:
			logger.Fatal("undefined converter", zap.String("name", convName))
		}
	}
	return reg
}
