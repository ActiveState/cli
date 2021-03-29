package server

import (
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/labstack/echo/v4"

	"github.com/ActiveState/cli/internal/logging"
)

const QuitRoute = "/__quit"

func (s *Server) setupRouting() {
	s.httpServer.POST("/query", func(c echo.Context) error {
		s.graphServer.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	s.httpServer.GET("/", func(c echo.Context) error {
		playground.Handler("GraphQL", "/query").ServeHTTP(c.Response(), c.Request())
		return nil
	})

	s.httpServer.GET(QuitRoute, func(c echo.Context) error {
		go func() {
			time.Sleep(100 * time.Millisecond) // give server time to respond
			err := s.Close()
			if err != nil {
				logging.Errorf("Closing server failed: %v", err)
			}
		}()
		return nil
	})
}
