package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/vpbuyanov/syncplay/internal/gen"
)

func (s *Server) GetInfo(ctx echo.Context) error {
	versionInfo := gen.GetInfo{
		Version: "0.0.1",
	}

	err := ctx.JSON(http.StatusOK, versionInfo)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Detail: "something wrong",
		})
	}

	return nil
}
