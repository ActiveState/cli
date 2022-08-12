package server

import (
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/labstack/echo/v4"
)

func (s *Server) setupRouting() {
	s.httpServer.POST("/query", func(c echo.Context) error {
		s.graphServer.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	s.httpServer.GET("/", func(c echo.Context) error {
		playground.Handler("GraphQL", "/query").ServeHTTP(c.Response(), c.Request())
		return nil
	})
}
