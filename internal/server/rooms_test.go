package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// TestServer_CreateRoom проверяет handler CreateRoom на успех и на ошибку модели.
func TestServer_CreateRoom(t *testing.T) {
	e := echo.New()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockModel := NewMockmodelRoom(ctrl)
	srv := &Server{m: mockModel}

	t.Run("успешное создание", func(t *testing.T) {
		mockModel.
			EXPECT().
			CreateRoom(gomock.Any()).
			Return("room-123", nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/rooms", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := srv.CreateRoom(c)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		expectedBody := `{"room_id":"room-123"}`
		assert.JSONEq(t, expectedBody, rec.Body.String())
		assert.Equal(t, echo.MIMEApplicationJSON, rec.Header().Get(echo.HeaderContentType))
	})

	t.Run("ошибка бизнес‑логики", func(t *testing.T) {
		mockModel.
			EXPECT().
			CreateRoom(gomock.Any()).
			Return("", errors.New("db failure"))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/rooms", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := srv.CreateRoom(c)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.JSONEq(t, `{"detail":"something wrong"}`, rec.Body.String())
		assert.Equal(t, echo.MIMEApplicationJSON, rec.Header().Get(echo.HeaderContentType))
	})
}

// TestServer_DeleteRoom проверяет handler DeleteRoom на успех и на ошибку модели.
func TestServer_DeleteRoom(t *testing.T) {
	e := echo.New()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockModel := NewMockmodelRoom(ctrl)
	srv := &Server{m: mockModel}

	id, err := uuid.NewUUID()
	assert.NoError(t, err)

	idStr := id.String()

	apiID := openapi_types.UUID{}
	err = apiID.Scan(idStr)
	assert.NoError(t, err)

	t.Run("успешное удаление", func(t *testing.T) {
		mockModel.
			EXPECT().
			DeleteRoom(gomock.Any(), apiID).
			Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/rooms/"+idStr, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(idStr)

		err := srv.DeleteRoom(c, apiID)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Empty(t, rec.Body.String())
	})

	t.Run("ошибка бизнес‑логики", func(t *testing.T) {
		mockModel.
			EXPECT().
			DeleteRoom(gomock.Any(), apiID).
			Return(errors.New("not found"))

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/rooms/"+idStr, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(idStr)

		err := srv.DeleteRoom(c, apiID)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.JSONEq(t, `{"detail":"something wrong"}`, rec.Body.String())
		assert.Equal(t, echo.MIMEApplicationJSON, rec.Header().Get(echo.HeaderContentType))
	})
}
