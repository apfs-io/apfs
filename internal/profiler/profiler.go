package profiler

import (
	"fmt"
	"net/http"

	"github.com/pkg/profile"
	"go.uber.org/zap"
)

// Run profiler
func Run(mode, listenAddr string, logger *zap.Logger) {
	switch mode {
	case "cpu":
		defer profile.Start(profile.CPUProfile).Stop()
	case "mem", "memory":
		defer profile.Start(profile.MemProfile).Stop()
	case "mutex":
		defer profile.Start(profile.MutexProfile).Stop()
	case "block":
		defer profile.Start(profile.BlockProfile).Stop()
	case "net":
		go func() {
			fmt.Printf("Run profile (port %s)\n", listenAddr)
			if err := http.ListenAndServe(listenAddr, nil); err != nil {
				logger.Error("profile server error", zap.Error(err))
			}
		}()
	}
}
