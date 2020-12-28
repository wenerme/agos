package apis

import (
	"encoding/json"
	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/quad"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/wenerme/agos/pkg/apki"
	"github.com/wenerme/agos/pkg/whoami"
	"github.com/wenerme/tools/pkg/apki/apis"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"time"
)

var router *mux.Router

func Router() *mux.Router {
	if router == nil {
		router = buildRouter()
	}
	return router
}
func buildRouter() *mux.Router {
	logrus.Info("build router")

	// logger, _ := zap.NewProduction()
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := config.Build()
	zap.ReplaceGlobals(logger)

	r := mux.NewRouter()
	r.Use(recoveryMiddleware)
	r.Use(loggingMiddleware)

	{
		r.HandleFunc("/api/ping", func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("PONG"))
		})
	}

	c := restful.NewContainer()
	cors := restful.CrossOriginResourceSharing{
		AllowedDomains: []string{"127.0.0.1:3000", "127.0.0.1:3001", "localhost:6006", "hoppscotch.io"},
		AllowedHeaders: []string{
			"Content-Type", "Accept", "Authorization",
		},
		ExposeHeaders:  []string{},
		AllowedMethods: []string{},
		Container:      c,
	}
	c.Filter(cors.Filter)
	c.Filter(c.OPTIONSFilter)
	{
		ws := new(restful.WebService)
		ws.Path("/whoami")
		ws.Route(ws.GET("").To(func(req *restful.Request, res *restful.Response) {
			whoami.Handler(res, req.Request)
		}))
		c.Add(ws)
	}
	ApkIndexResource{}.RegisterTo(c)

	r.PathPrefix("/api/v1").Handler(http.StripPrefix("/api/v1", c))
	return r
}

type ApkIndexResource struct {
}

func (svc ApkIndexResource) RegisterTo(container *restful.Container) {

	ws := new(restful.WebService)
	ws.Path("/apki").Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/stats").To(svc.Stats))
	ws.Route(ws.GET("/packages/{package}/graph").To(svc.GetPackageGraph))

	container.Add(ws)
}

func (svc ApkIndexResource) Stats(req *restful.Request, res *restful.Response) {
	g, err := apki.GetGraph()
	throwError(err, "GetGraph")
	stats, err := g.Store.Stats(nil, false)
	throwError(err, "Stats")
	swallowError(res.WriteEntity(ApkiStats{
		GraphStats: stats,
	}), "WriteEntity")
}

type ApkiStats struct {
	GraphStats graph.Stats
}

func (svc ApkIndexResource) GetPackageGraph(req *restful.Request, res *restful.Response) {
	pkg := req.PathParameter("package")

	g, err := apki.GetGraph()
	throwError(err, "GetGraph")

	p := cayley.StartPath(g.Store, quad.String(pkg)).Out(quad.String("depend")).In("provide")
	all, err := p.Iterate(nil).AllValues(nil)
	throwError(err, "Query Graph")
	result := &Relationship{Dependencies: all}

	swallowError(res.WriteEntity(result), "WriteEntity")
}

type Relationship struct {
	Dependencies interface{} `json:"dependencies"`
}

const mimeJSON = "application/json;charset=utf-8"

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Content-Type", mimeJSON)
				ee := apis.FromError(err)
				w.WriteHeader(ee.Status)
				zap.L().Error(ee.Message, zap.Int("status", ee.Status), zap.String("reason", ee.Reason), zap.String("code", ee.Code), zap.String("uri", r.RequestURI))
				e := json.NewEncoder(w).Encode(ee)
				if e != nil {
					zap.S().With("error", e).Warn("marshal recovery error failed")
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}

var requestID = atomic.NewInt64(0)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := requestID.Add(1)
		start := time.Now()
		zap.L().Info("->", zap.Int64("id", id), zap.String("remote", r.RemoteAddr), zap.String("method", r.Method), zap.String("uri", r.RequestURI))
		next.ServeHTTP(w, r)
		esp := time.Since(start)
		zap.L().Info("<-", zap.Int64("id", id), zap.String("uri", r.RequestURI), zap.Int64("time", esp.Milliseconds()), zap.String("content-type", w.Header().Get("content-type")))
	})
}
