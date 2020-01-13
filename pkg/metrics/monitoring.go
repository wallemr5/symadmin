package metrics

import (
	"sync"

	"fmt"

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
)

type OcPrometheus struct {
	Exporter *ocprom.Exporter
	Router   *gin.Engine
	Mx       sync.Mutex
}

func NewOcPrometheus(router *gin.Engine) (*OcPrometheus, error) {
	registry := prometheus.DefaultRegisterer.(*prometheus.Registry)
	p := &OcPrometheus{
		Exporter: nil,
		Router:   nil,
	}
	exporter, err := ocprom.NewExporter(ocprom.Options{Registry: registry})
	if err != nil {
		err = fmt.Errorf("could not set up prometheus exporter: %v", err)
		return nil, err
	}

	p.Exporter = exporter
	p.Router = router
	view.RegisterExporter(exporter)

	// Register stat views
	err = view.Register(
		// Gin (HTTP) stats
		ochttp.ServerRequestCountView,
		ochttp.ServerRequestBytesView,
		ochttp.ServerResponseBytesView,
		ochttp.ServerLatencyView,
		ochttp.ServerRequestCountByMethod,
		ochttp.ServerResponseCountByStatusCode,
	)
	if err != nil {
		panic(err)
	}

	return p, nil
}
