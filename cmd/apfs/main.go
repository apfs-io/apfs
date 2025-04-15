//
// @project apfs 2017 - 2022, 2025
// @author Dmitry Ponomarev <demdxx@gmail.com> 2017 - 2022, 2025
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
	"os/signal"

	"github.com/demdxx/goconfig"
	"go.uber.org/zap"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/apfs-io/apfs/cmd/apfs/appcontext"
	"github.com/apfs-io/apfs/cmd/apfs/commands"
	_ "github.com/apfs-io/apfs/cmd/apfs/dbinit"
	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	"github.com/apfs-io/apfs/internal/context/version"
	"github.com/apfs-io/apfs/internal/profiler"
	"github.com/apfs-io/apfs/internal/zlogger"
)

var (
	config       appcontext.ConfigType
	buildCommit  = ""
	buildVersion = "develop"
	buildDate    = "unknown"
)

// Define command list
var cmdList = commands.ICommands{
	commands.ServerCommand,
	commands.ProcessorCommand,
}

func init() {
	fmt.Println()
	fmt.Println("█████╗ ██████╗ ███████╗███████╗")
	fmt.Println("██╔══██╗██╔══██╗██╔════╝██╔════╝")
	fmt.Println("███████║██████╔╝█████╗  ███████╗")
	fmt.Println("██╔══██║██╔═══╝ ██╔══╝  ╚════██║")
	fmt.Println("██║  ██║██║     ██║     ███████║")
	fmt.Println("╚═╝  ╚═╝╚═╝     ╚═╝     ╚══════╝")
	fmt.Println()
	fmt.Println("Version:", buildVersion, " (", buildCommit, ")")
	fmt.Println("Build date:", buildDate)
	fmt.Println()

	args := os.Args
	if len(args) > 1 {
		args = args[2:]
	}

	fatalError(goconfig.Load(
		&config,
		goconfig.WithDefaults(),
		goconfig.WithCustomArgs(args...),
		goconfig.WithEnv(),
	), "config loading")

	// Init new logger object
	loggerObj, err := zlogger.New(config.ServiceName, config.LogEncoder,
		config.LogLevel, config.LogAddr, zap.Fields(
			zap.String("commit", buildCommit),
			zap.String("version", buildVersion),
			zap.String("build_date", buildDate),
		))
	fatalError(err, "configure logger")

	// Replace global logger
	zap.ReplaceGlobals(loggerObj)

	// Print configuration
	if config.IsDebug() {
		fmt.Println(config.String())
	}
}

func main() {
	var (
		logger      = zap.L()
		ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt)
	)
	defer cancel()

	// Add logger to context
	ctx = ctxlogger.WithLogger(ctx, logger)

	// Register version information
	ctx = version.WithContext(ctx, &version.Version{
		Commit:  buildCommit,
		Version: buildVersion,
		Date:    buildDate,
	})

	if len(os.Args) < 2 {
		printCommandsUsage()
		return
	}

	// Get command name
	cmdName := os.Args[1]

	// Run command by name
	icmd := cmdList.Get(cmdName)

	// Print help if command not found
	if cmdName == "help" || icmd == nil {
		printCommandsUsage()
		return
	}

	// Profiling server of collector
	profiler.Run(config.Server.Profile.Mode,
		config.Server.Profile.Listen, logger)

	// Run command with context
	fmt.Println()
	fmt.Println("░█ Run command:\x1b[31m", icmd.Cmd(), "\x1b[0m")
	fmt.Println()

	fatalError(icmd.Run(ctx, os.Args[2:]), "command execution")
}

func printCommandsUsage() {
	fmt.Println("Usage: apfs <command> [options]")
	fmt.Println("Commands:")
	for _, cmd := range cmdList {
		fmt.Printf("  % 10s - %s\n", cmd.Cmd(), cmd.Help())
	}
	fmt.Println("  help - print this help")
}

func fatalError(err error, msgs ...any) {
	if err != nil {
		log.Fatalln(append(msgs, err)...)
	}
}
