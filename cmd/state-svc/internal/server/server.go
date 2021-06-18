package server

import (
	"context"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cast"

	"github.com/ActiveState/cli/cmd/state-svc/internal/resolver"
	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type Server struct {
	cfg         *config.Instance
	shutdown    chan<- struct{}
	graphServer *handler.Server
	listener    net.Listener
	httpServer  *echo.Echo
	port        int
}

func New(cfg *config.Instance, shutdown chan<- struct{}) (*Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to listen")
	}

	s := &Server{cfg: cfg, shutdown: shutdown}
	s.graphServer = newGraphServer(cfg)
	s.listener = listener
	s.httpServer = newHTTPServer(listener)

	s.setupRouting()

	_, portEncoded, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse port from address: %v", s.listener.Addr().String())
	}
	s.port, err = strconv.Atoi(portEncoded)
	if err != nil {
		return nil, errs.Wrap(err, "Could not convert port: %v", portEncoded)
	}

	return s, nil
}

func (s *Server) Port() int {
	return s.port
}

func (s *Server) Start() error {
	return s.httpServer.Start(s.listener.Addr().String())
}

func (s *Server) Shutdown() error {
	logging.Debug("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return errs.Wrap(err, "Could not close http server")
	}

	err := s.cfg.SetWithLock(constants.SvcConfigPid, func(setPidI interface{}) (interface{}, error) {
		setPid := cast.ToInt(setPidI)
		if setPid != os.Getpid() {
			return nil, errs.New("PID in configuration file does not match PID of server shutting down")
		}
		return "", nil
	})
	if err != nil {
		return errs.Wrap(err, "Could not unset State Service PID in configuration file.")
	}
	return nil
}

func newGraphServer(cfg *config.Instance) *handler.Server {
	graphServer := handler.NewDefaultServer(genserver.NewExecutableSchema(genserver.Config{Resolvers: resolver.New(cfg)}))
	graphServer.AddTransport(&transport.Websocket{})
	graphServer.SetQueryCache(lru.New(1000))
	graphServer.Use(extension.Introspection{})
	graphServer.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})
	return graphServer
}

func newHTTPServer(listener net.Listener) *echo.Echo {
	httpServer := echo.New()
	httpServer.Listener = listener
	httpServer.Use(middleware.Logger())
	httpServer.Use(middleware.Recover())
	return httpServer
}
