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
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/ActiveState/cli/cmd/state-svc/internal/resolver"
	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
)

type Server struct {
	graphServer *handler.Server
	listener    net.Listener
	httpServer  *echo.Echo
	port        int
}

func New(cfg *config.Instance) (*Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to listen")
	}

	s := &Server{}
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return errs.Wrap(err, "Could not close http server")
	}
	if err := s.listener.Close(); err != nil {
		return errs.Wrap(err, "Could not close listener")
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
