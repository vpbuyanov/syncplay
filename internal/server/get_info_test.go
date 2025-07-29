package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestServer_GetInfo(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &Server{}
	err := s.GetInfo(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get(echo.HeaderContentType))
	assert.JSONEq(t, `{"version":"0.0.1"}`, rec.Body.String())
}
