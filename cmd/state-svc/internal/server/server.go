package server

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/ActiveState/cli/internal/analytics/client/sync"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/ActiveState/cli/cmd/state-svc/internal/deprecation"
	"github.com/ActiveState/cli/cmd/state-svc/internal/resolver"
	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	"github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type Server struct {
	shutdown    context.CancelFunc
	resolver    *resolver.Resolver
	graphServer *handler.Server
	listener    net.Listener
	httpServer  *echo.Echo
	port        int
	analytics   *sync.Client
}

func New(cfg *config.Instance, an *sync.Client, checker *deprecation.Checker, shutdown context.CancelFunc) (*Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to listen")
	}

	s := &Server{shutdown: shutdown, resolver: resolver.New(cfg, an, checker), analytics: an}

	s.graphServer = newGraphServer(s.resolver)
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
	s.analytics.Event(constants.CatStateSvc, "start")
	err := s.httpServer.Start(s.listener.Addr().String())
	if err != nil {
		s.analytics.Event(constants.CatStateSvc, "start-failure")
	}
	return err
}

func (s *Server) Shutdown() error {
	s.analytics.Event(constants.CatStateSvc, "shutdown")
	logging.Debug("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return errs.Wrap(err, "Could not close http server")
	}
	if err := s.resolver.Close(); err != nil {
		return errs.Wrap(err, "Could not close resolver")
	}

	return nil
}

func newGraphServer(r *resolver.Resolver) *handler.Server {
	graphServer := handler.NewDefaultServer(genserver.NewExecutableSchema(genserver.Config{Resolvers: r}))
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
