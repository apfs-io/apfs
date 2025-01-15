//
// @project apfs 2017 - 2022
// @author Dmitry Ponomarev <demdxx@gmail.com> 2017 - 2022
//

package main

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"runtime"
	"strings"

	"go.uber.org/zap"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/geniusrabbit/notificationcenter/v2/wrappers/concurrency"
	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/cmd/apfs/appcontext"
	"github.com/apfs-io/apfs/cmd/apfs/appinit"
	_ "github.com/apfs-io/apfs/cmd/apfs/dbinit"
	"github.com/apfs-io/apfs/cmd/apfs/server"
	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	"github.com/apfs-io/apfs/internal/profiler"
	v1 "github.com/apfs-io/apfs/internal/server/v1"
	"github.com/apfs-io/apfs/internal/stream"
)

var (
	commit     = ""
	appVersion = "develop"
)

const (
	eventStream      = appinit.EventStreamName
	commandServer    = "server"
	commandProcessor = "processor"
)

func main() {
	var command string
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		command = os.Args[1]
		os.Args = os.Args[1:]
	}

	conf := &appcontext.Config
	fatalError(conf.Load(), "load config:")

	if conf.IsDebug() {
		fmt.Println(" ⇛ DEBUG mode ON")
		fmt.Println("config", conf)
	}

	if conf.Storage.Connect == `` {
		fatalError(fmt.Errorf("storage connect is not defined"))
	}

	// Init new logger object
	var (
		ctx         = context.Background()
		logger, err = newLogger(conf.IsDebug())
	)
	zap.ReplaceGlobals(logger)
	fatalError(err, "logger")

	// Init server object
	protoAPI, err := appinit.ProtocolAPIObject(ctx, conf, logger)
	fatalError(err, "server object")
	srvAPI := v1.NewHTTPWrapper(protoAPI)

	switch command {
	case commandServer:
		profiler.Run(conf.Server.Profile.Mode, conf.Server.Profile.Listen, logger)
		if conf.Processing {
			go func() {
				fatalError(runProcessor(ctx, conf, protoAPI.(nc.Receiver), logger), "processor")
			}()
		}
		fatalError(runServer(ctx, conf, srvAPI, logger), "server")
	case commandProcessor:
		profiler.Run(conf.Server.Profile.Mode, conf.Server.Profile.Listen, logger)
		fatalError(runProcessor(ctx, conf, protoAPI.(nc.Receiver), logger), "processor")
	default:
		fmt.Println("Undefined")
	}
}

func runServer(ctx context.Context, conf *appcontext.ConfigType, api *v1.ServerHTTPWrapper, logger *zap.Logger) error {
	fmt.Println(" ⇛ Run apfs gRPC server")
	fmt.Println(" ⇛ storage connect:", conf.Storage.Connect)

	ctxWrp := func(context.Context) context.Context {
		return ctxlogger.WithLogger(ctx, logger)
	}
	srv := &server.GRPCServer{
		API:               api,
		ContextWrap:       ctxWrp,
		Logger:            logger,
		Concurrency:       conf.Server.GRPC.Concurrency,
		RequestTimeout:    conf.Server.GRPC.Timeout,
		ConnectionTimeout: conf.Server.GRPC.Timeout,
		CertFile:          "",
		KeyFile:           "",
	}
	if conf.Server.GRPC.Listen != "" && conf.Server.HTTP.Listen != "" {
		go func() {
			if err := srv.RunGRPC(ctx, conf.Server.GRPC.Listen); err != nil {
				panic(err)
			}
		}()
	} else if conf.Server.GRPC.Listen != "" {
		return srv.RunGRPC(ctx, conf.Server.GRPC.Listen)
	}
	return srv.RunHTTP(ctx, conf.Server.HTTP.Listen)
}

func runProcessor(ctx context.Context, conf *appcontext.ConfigType, reveiver nc.Receiver, logger *zap.Logger) error {
	fmt.Println(" ⇛ Run apfs file processor")
	fmt.Println("storage connect:", conf.Storage.Connect)

	// Register the notification stream
	events, err := stream.NewReader(ctx, conf.Eventstream.Connect)
	if err != nil {
		return errors.Wrap(err, "connect to: "+conf.Eventstream.Connect)
	}
	if err = nc.Register(eventStream, events); err != nil {
		return err
	}
	if conf.Eventstream.Concurrency <= 0 {
		conf.Eventstream.Concurrency = runtime.NumCPU()
	}

	// Concurrent pool wrapper
	conncurrencyReceiver := concurrency.WithWorkers(
		reveiver, conf.Eventstream.Concurrency, nil,
		concurrency.WithWorkerPoolSize(conf.Eventstream.PoolSize),
		concurrency.WithRecoverHandler(func(err any) {
			switch e := err.(type) {
			case error:
				logger.Error(`processing pool recovery`, zap.Error(e))
			default:
				logger.Error(`processing pool recovery`, zap.Any(`error`, err))
			}
		}),
	)
	if err = nc.Subscribe(ctx, eventStream, conncurrencyReceiver); err != nil {
		return errors.Wrap(err, "subscribe processor handler")
	}
	fmt.Println("Run listener:", conf.Eventstream.Connect)
	return nc.Listen(ctx)
}

func newLogger(debug bool) (*zap.Logger, error) {
	loggerFields := zap.Fields(
		zap.String("commit", commit),
		zap.String("version", appVersion),
	)
	if debug {
		return zap.NewDevelopment(loggerFields)
	}
	return zap.NewProduction(loggerFields)
}

func fatalError(err error, msgs ...any) {
	if err != nil {
		log.Fatalln(append(msgs, err)...)
	}
}
