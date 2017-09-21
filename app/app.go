package app

import (
	"html/template"
	"net/http"
	"strings"

	stdlog "log"

	"github.com/prasannavl/go-gluons/http/fileserver"
	"github.com/prasannavl/go-gluons/http/hostrouter"
	"github.com/prasannavl/go-gluons/http/httpservice"
	"github.com/prasannavl/go-gluons/http/middleware"
	"github.com/prasannavl/go-gluons/http/reverseproxy"
	"github.com/prasannavl/go-gluons/log"
	"github.com/prasannavl/mchain"
	"github.com/prasannavl/mchain/hconv"
	"github.com/prasannavl/mroute"
	"github.com/prasannavl/mroute/pat"
)

func newAppHandler(c *AppContext, webRoot string) mchain.Handler {
	router := mroute.NewMux()

	router.Use(
		middleware.InitMiddleware(c.Logger),
		middleware.LoggerMiddleware(log.InfoLevel),
		middleware.ErrorHandlerMiddleware,
		middleware.PanicRecoveryMiddleware,
		middleware.RequestIDMiddleware(false),
	)

	dir := http.Dir(webRoot)
	router.Handle(pat.New("/*"), fileserver.NewEx(dir, nil))

	return router
}

func createAppContext(logger *log.Logger, addr string) *AppContext {
	services := Services{
		Logger:        logger,
		TemplateCache: make(map[string]*template.Template),
	}
	c := AppContext{
		Services:      services,
		ServerAddress: addr,
	}
	return &c
}

func NewApp(logger *log.Logger, addr string, webRoot string, hosts []string, stdLogger *stdlog.Logger) http.Handler {
	context := createAppContext(logger, addr)
	appHandler := newAppHandler(context, webRoot)
	if len(hosts) == 0 {
		return hconv.ToHttp(appHandler, nil)
	}
	r := hostrouter.New()
	log.Infof("host filters: %v", hosts)

	for _, h := range hosts {
		// TODO: Later, extract this to config, may be.
		if strings.HasSuffix(h, "statwick.com") {
			const statwickHost = "localhost:8080"
			proxy := reverseproxy.NewHostProxy(statwickHost, true)
			proxy.ErrorLog = stdLogger
			r.HandlePattern(h, hconv.FromHttp(proxy, false))
		} else {
			r.HandlePattern(h, appHandler)
		}
	}
	return r.BuildHttp(nil)
}

func CreateService(opts *httpservice.HandlerServiceOpts, stdLogger *stdlog.Logger) (httpservice.Service, error) {
	app := NewApp(opts.Logger, opts.Addr, opts.WebRoot, opts.Hosts, stdLogger)
	opts.Handler = app
	return httpservice.NewHandlerService(opts)
}
