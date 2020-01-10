package router

import (
	"net/http"
	"text/template"

	"context"
	"crypto/tls"
	"time"

	"bytes"

	"fmt"

	"github.com/DeanThompson/ginpprof"
	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/metrics"
	"gitlab.dmall.com/arch/sym-admin/pkg/version"
	"k8s.io/klog"
)

const (
	VersionPath = "version"
	MetricsPath = "/metrics"
	LivePath    = "/live"
	ReadyPath   = "/ready"
	PprofPath   = "/debug/pprof"
)

// RouterOptions are options for constructing a Router
type RouterOptions struct {
	GinLogEnabled  bool
	PprofEnabled   bool
	MetricsEnabled bool

	Addr             string
	MetricsSubsystem string
	MetricsPath      string

	// 	Username      string
	// 	Password      string
	CertFilePath string
	KeyFilePath  string
}

// Router handles all incoming HTTP requests
type Router struct {
	*gin.Engine
	Routes              map[string][]*Route
	Addr                string
	httpServer          *http.Server
	CertFilePath        string
	KeyFilePath         string
	ProfileDescriptions []*Profile
	ShutdownTimeout     time.Duration
}

type Profile struct {
	Name string
	Href string
	Desc string
}

// Route represents an application route
type Route struct {
	Method  string
	Path    string
	Handler gin.HandlerFunc
	Desc    string
}

// NewRouter creates a new Router instance
func NewRouter(opt *RouterOptions) *Router {
	engine := gin.New()
	engine.Use(gin.Recovery())
	// engine := gin.Default()
	// engine.Use(limits.RequestSizeLimiter(int64(opt.MaxUploadSize)))
	if !opt.GinLogEnabled {
		gin.SetMode(gin.ReleaseMode)
	} else {
		engine.Use(gin.Logger())
		// engine.Use(ginlog.Middleware())
	}

	r := &Router{
		Engine:              engine,
		Routes:              make(map[string][]*Route, 0),
		ProfileDescriptions: make([]*Profile, 0),
	}

	if opt.PprofEnabled {
		// automatically add routers for net/http/pprof e.g. /debug/pprof, /debug/pprof/heap, etc.
		ginpprof.Wrap(r.Engine)
		r.AddProfile(PprofPath, `PProf related things:<br/>
			<a href="/debug/pprof/goroutine?debug=2">full goroutine stack dump</a>`)
	}

	if opt.MetricsEnabled {
		klog.Infof("start load router path:%s ", opt.MetricsPath)
		p := metrics.NewPrometheus(opt.MetricsSubsystem, []string{})
		p.Use(r.Engine, opt.MetricsPath)
		r.AddProfile(MetricsPath, "Prometheus format metrics")
	}

	r.CertFilePath = opt.CertFilePath
	r.KeyFilePath = opt.KeyFilePath
	r.Addr = opt.Addr
	r.NoRoute(r.masterHandler)
	return r
}

func (r *Router) Start(stopCh <-chan struct{}) error {
	if r.ShutdownTimeout == 0 {
		r.ShutdownTimeout = 5 * time.Second
	}

	r.httpServer = &http.Server{
		Addr:         r.Addr,
		Handler:      r.Engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	if r.CertFilePath != "" && r.KeyFilePath != "" {
		cert, err := tls.LoadX509KeyPair(r.CertFilePath, r.KeyFilePath)
		if err != nil {
			klog.Errorf("LoadX509KeyPair err:%+v", err)
			return err
		}
		r.httpServer.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	errCh := make(chan error)
	go func() {
		if r.CertFilePath != "" && r.KeyFilePath != "" {
			klog.Infof("Listening on %s, https://localhost%s\n", r.Addr, r.Addr)
			if err := r.httpServer.ListenAndServeTLS(r.CertFilePath, r.KeyFilePath); err != nil && err != http.ErrServerClosed {
				klog.Error("Https server error: ", err)
				errCh <- err
			}
		} else {
			klog.Infof("Listening on %s, http://localhost%s\n", r.Addr, r.Addr)
			if err := r.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				klog.Error("Http server error: ", err)
				errCh <- err
			}
		}
	}()

	var err error
	select {
	case <-stopCh:
		klog.Infof("Shutting down the http/https:%s server...", r.Addr)
		if r.ShutdownTimeout > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), r.ShutdownTimeout)
			defer cancel()
			err = r.httpServer.Shutdown(ctx)
		} else {
			err = r.httpServer.Close()
		}
	case err = <-errCh:
	}

	if err != nil {
		klog.Fatalf("Server stop err: %#v", err)
	} else {
		klog.Infof("Server exiting")
	}

	return err
}

func (r *Router) StartWarp(stopCh <-chan struct{}) {
	_ = r.Start(stopCh)
}

func (r *Router) AddProfile(href, desc string) {
	r.ProfileDescriptions = append(r.ProfileDescriptions, &Profile{
		Name: href,
		Href: href,
		Desc: desc,
	})
}

// SetRoutes applies list of routes
func (r *Router) AddRoutes(apiGroup string, routes []*Route) {
	klog.V(3).Infof("load apiGroup:%s", apiGroup)
	for _, route := range routes {
		switch route.Method {
		case "GET":
			r.GET(route.Path, route.Handler)
		case "POST":
			r.POST(route.Path, route.Handler)
		case "DELETE":
			r.DELETE(route.Path, route.Handler)
		case "Any":
			r.Any(route.Path, route.Handler)
		default:
			klog.Warningf("no method:%s apiGroup:%s", route.Method, apiGroup)
		}
	}

	if rs, ok := r.Routes[apiGroup]; !ok {
		r.Routes[apiGroup] = routes
	} else {
		rs = append(rs, routes...)
	}

	if apiGroup == "health" {
		r.AddProfile(LivePath, `liveness check: <br/>
			<a href="/live?full=true"> query the full body`)
		r.AddProfile(ReadyPath, `readyness check:  <br/>
			<a href="/ready?full=true"> query the full body`)
	} else if apiGroup == "cluster" {
		for _, route := range routes {
			var desc string
			if route.Desc == "" {
				desc = fmt.Sprintf("name: the unique cluster name and all <br/> appName: the unique app name")
			} else {
				desc = route.Desc
			}
			r.AddProfile(route.Path, desc)
		}
	}
}

// all incoming requests are passed through this handler
func (r *Router) masterHandler(c *gin.Context) {
	klog.V(4).Infof("no router for method:%s, url:%s", c.Request.Method, c.Request.URL.Path)
	c.JSON(404, gin.H{
		"Method": c.Request.Method,
		"Path":   c.Request.URL.Path,
		"error":  "router not found"})
}

// IndexHandler
func (r *Router) IndexHandler(c *gin.Context) {
	var b bytes.Buffer
	indexTmpl.Execute(&b, r.ProfileDescriptions)
	c.Data(http.StatusOK, "", b.Bytes())
}

// LiveHandler
func LiveHandler(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

// ReadHandler
func ReadHandler(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

// VersionHandler
func VersionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, version.GetVersion())
}

// DefaultRoutes
func (r *Router) DefaultRoutes() []*Route {
	var routes []*Route

	appRoutes := []*Route{
		{"GET", "/", r.IndexHandler, ""},
		{"GET", VersionPath, VersionHandler, ""},
	}

	routes = append(routes, appRoutes...)
	return routes
}

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html><html>
<head>
<title>/debug/pprof/</title>
<style>
.profile-name{
	display:inline-block;
	width:6rem;
}
</style>
</head>
<body>
Things to do:
{{range .}}
<h2><a href={{.Href}}>{{.Name}}</a></h2>
<p>
{{.Desc}}
</p>
{{end}}
</body>
</html>
`))
