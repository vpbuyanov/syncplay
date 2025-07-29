package server

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/pkg/errors"

	"github.com/vpbuyanov/syncplay/internal/config"
	"github.com/vpbuyanov/syncplay/internal/gen"
)

//go:generate mockgen -source=server.go -destination server_mock.go -package server SERVER
type modelRoom interface {
	CreateRoom(ctx context.Context) (string, error)
	DeleteRoom(ctx context.Context, id openapi_types.UUID) error
}

type Server struct {
	e *echo.Echo
	m modelRoom
}

func NewServer(cfg config.Server, m modelRoom) (*Server, error) {
	server := &Server{
		e: echo.New(),
		m: m,
	}

	server.e.HideBanner = true
	server.e.Pre(middleware.RemoveTrailingSlash())
	server.e.Use(middleware.Recover())
	server.e.Use(
		middleware.CORS(),
	)
	server.e.Pre(
		middleware.TimeoutWithConfig(
			middleware.TimeoutConfig{
				Timeout: cfg.TimeOut,
			},
		),
	)

	gen.RegisterHandlers(server.e, server)

	server.e.Server.Addr = cfg.String()

	return server, nil
}

func (s *Server) Listen() error {
	if err := s.e.StartServer(s.e.Server); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.e.Logger.Fatal("start: " + err.Error())
	}

	return nil
}
