package server

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/ActiveState/cli/cmd/state-svc/internal/resolver"
	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type Server struct {
	cancel      context.CancelFunc
	done        chan bool
	graphServer *handler.Server
	listener    net.Listener
	httpServer  *echo.Echo
	port        int
}

func New(cfg *config.Instance, cancel context.CancelFunc) (*Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to listen")
	}

	s := &Server{cancel: cancel}
	s.done = make(chan bool)
	s.graphServer = newGraphServer(cfg, s.done)
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

func (s *Server) quit() {
	s.done <- true
	close(s.done)
	s.cancel()
}

func (s *Server) Shutdown() error {
	logging.Debug("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return errs.Wrap(err, "Could not close http server")
	}

	return nil
}

func newGraphServer(cfg *config.Instance, done chan bool) *handler.Server {
	graphServer := handler.NewDefaultServer(genserver.NewExecutableSchema(genserver.Config{Resolvers: resolver.New(cfg, done)}))
	graphServer.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// For development. User proper CORS for prod
				return true
			},
		},
		KeepAlivePingInterval: 10 * time.Second,
	})
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
