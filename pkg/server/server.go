package server

import (
	"fmt"
	slog "log"
	"net/http"

	"git.containerum.net/ch/api-gateway/pkg/model"
	middle "git.containerum.net/ch/api-gateway/pkg/server/middleware"
	"git.containerum.net/ch/auth/proto"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/gonic"
	"git.containerum.net/ch/kube-client/pkg/cherry/api-gateway"
	"github.com/gin-gonic/gin"

	"net"

	log "github.com/sirupsen/logrus"
)

//Server keeps HTTP sever and it configs
type Server struct {
	http.Server
	options *ServerOptions
}

type ServerOptions struct {
	Routes  *model.Routes
	Config  *model.Config
	Auth    *authProto.AuthClient
	Metrics *model.Metrics
}

//New return configurated server with all handlers
func New(opt *ServerOptions) (*Server, error) {
	handlers, err := createHandler(opt)
	if err != nil {
		return nil, err
	}
	return &Server{
		options: opt,
		Server: http.Server{
			Addr:     fmt.Sprintf("0.0.0.0:%v", opt.Config.Port),
			Handler:  handlers,
			ErrorLog: slog.New(log.New().Writer(), "server", 0),
		},
	}, nil
}

//Start return http or https ListenServer
func (s *Server) Start() error {
	s.ConnState = func(c net.Conn, st http.ConnState) {
		log.Info(fmt.Sprintf("ConnState: %v\n", st.String()))
	}
	if s.options.Config.TLS.Enable {
		return s.Server.ListenAndServeTLS(s.options.Config.TLS.Cert, s.options.Config.TLS.Key)
	}
	return s.ListenAndServe()
}

func createHandler(opt *ServerOptions) (http.Handler, error) {
	router := gin.New()
	limiter := middle.CreateLimiter(opt.Config.Rate.Limit)
	//Add middlewares
	router.Use(gonic.Recovery(gatewayErrors.ErrInternal, cherrylog.NewLogrusAdapter(log.WithField("component", "gin_recovery"))))
	router.Use(middle.Logger(opt.Metrics), middle.Cors())
	router.Use(limiter.Limit())
	router.Use(middle.SetHeaderFromQuery(), middle.ClearXHeaders(), middle.SetRequestID())
	router.Use(middle.CheckUserClientHeader(), middle.SetMainUserXHeaders())
	//Add routes
	for _, route := range opt.Routes.Routes {
		if !route.Active {
			continue
		}
		if opt.Config.Auth.Enable {
			router.Handle(route.Method, route.Listen, middle.SetRequestName(route.ID), middle.CheckAuth(route.Roles, opt.Auth), proxyHandler(route))
		} else {
			router.Handle(route.Method, route.Listen, middle.SetRequestName(route.ID), proxyHandler(route))
		}
	}
	return router, nil
}
