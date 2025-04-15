package tools

import (
	"net/http"
	"strconv"
	"text/template"

	swaggerFiles "github.com/swaggo/files"
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
)

type swaggerUIBundle struct {
	URL         string
	DeepLinking bool
}

// SwaggerServer returns swagger specification files located under "/swagger/"
func SwaggerServer(url string, deepLinking bool) http.HandlerFunc {
	//create a template with name
	t := template.New("swagger_index.html")
	index, _ := t.Parse(swaggerIndexTempl)
	jsonLen := strconv.Itoa(len(protocol.APIDocsSwaggerJSON))

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "index.html", "":
			err := index.Execute(w, &swaggerUIBundle{
				URL:         url,
				DeepLinking: deepLinking,
			})
			if err != nil {
				ctxlogger.Get(r.Context()).
					Error("write HTTP template response", zap.Error(err))
			}
		case "swagger.json":
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", jsonLen)
			_, err := w.Write([]byte(protocol.APIDocsSwaggerJSON))
			if err != nil {
				ctxlogger.Get(r.Context()).
					Error("write HTTP response", zap.Error(err))
			}
		default:
			swaggerFiles.Handler.ServeHTTP(w, r)
		}
	}
}
