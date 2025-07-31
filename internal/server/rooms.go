package server

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/vpbuyanov/syncplay/internal/gen"
)

func (s *Server) CreateRoom(ctx echo.Context) error {
	id, err := s.m.CreateRoom(ctx.Request().Context())
	if err != nil {
		slog.Error("Msg Err", "err", err)

		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Detail: "something wrong",
		})
	}

	res := gen.CreateRoom{
		RoomId: id,
	}

	return ctx.JSON(http.StatusOK, res)
}

func (s *Server) DeleteRoom(ctx echo.Context, id openapi_types.UUID) error {
	err := s.m.DeleteRoom(ctx.Request().Context(), id)
	if err != nil {
		slog.Error("Msg Err", "err", err)

		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Detail: "something wrong",
		})
	}

	return ctx.NoContent(http.StatusNoContent)
}
