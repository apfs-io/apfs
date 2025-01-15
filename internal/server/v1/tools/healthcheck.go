package tools

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
)

// HealthCheck of service
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"status":"OK"}`)); err != nil {
		ctxlogger.Get(r.Context()).Error("write HTTP response", zap.Error(err))
	}
}
