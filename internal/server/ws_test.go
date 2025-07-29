package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vpbuyanov/syncplay/internal/gen"
	"github.com/vpbuyanov/syncplay/internal/server"
)

func setupEcho() *echo.Echo {
	e := echo.New()
	srv := &server.Server{}

	e.GET("/ws/:id", func(c echo.Context) error {
		// разбираем path-параметр
		idStr := c.Param("id")
		uid, err := uuid.Parse(idStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Detail: "invalid id"})
		}
		openapiID := openapi_types.UUID(uid)
		return srv.ConnectRoomWS(c, openapiID)
	})

	return e
}

func TestConnectRoomWS_RoleAssignment(t *testing.T) {
	e := setupEcho()
	ts := httptest.NewServer(e)
	defer ts.Close()

	// генерируем UUID комнаты
	roomID := uuid.NewString()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/" + roomID

	ws1, resp, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer ws1.Close()
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	var roleMsg map[string]string
	err = ws1.ReadJSON(&roleMsg)
	require.NoError(t, err)
	assert.Equal(t, "host", roleMsg["role"], "Первый клиент должен стать host")

	ws2, resp, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer ws2.Close()
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	err = ws2.ReadJSON(&roleMsg)
	require.NoError(t, err)
	assert.Equal(t, "peer", roleMsg["role"], "Второй клиент должен стать peer")
}

func TestConnectRoomWS_MessageRelay(t *testing.T) {
	e := setupEcho()
	ts := httptest.NewServer(e)
	defer ts.Close()

	roomID := uuid.NewString()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/" + roomID

	hostConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer hostConn.Close()
	_ = hostConn.ReadJSON(new(map[string]string))

	peerConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer peerConn.Close()
	_ = peerConn.ReadJSON(new(map[string]string))

	offer := server.Message{
		Type:    "offer",
		Payload: json.RawMessage(`{"sdp":"dummy-offer"}`),
	}
	err = hostConn.WriteJSON(offer)
	require.NoError(t, err)

	var recv server.Message
	err = peerConn.ReadJSON(&recv)
	require.NoError(t, err)
	assert.Equal(t, offer.Type, recv.Type)
	assert.JSONEq(t, string(offer.Payload), string(recv.Payload))

	answer := server.Message{
		Type:    "answer",
		Payload: json.RawMessage(`{"sdp":"dummy-answer"}`),
	}
	err = peerConn.WriteJSON(answer)
	require.NoError(t, err)

	var recv2 server.Message
	err = hostConn.ReadJSON(&recv2)
	require.NoError(t, err)
	assert.Equal(t, answer.Type, recv2.Type)
	assert.JSONEq(t, string(answer.Payload), string(recv2.Payload))
}

func TestConnectRoomWS_HostDisconnects_CleansUp(t *testing.T) {
	e := setupEcho()
	ts := httptest.NewServer(e)
	defer ts.Close()

	roomID := uuid.NewString()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/" + roomID

	hostConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	_ = hostConn.ReadJSON(new(map[string]string))

	peerConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	_ = peerConn.ReadJSON(new(map[string]string))

	require.NoError(t, hostConn.Close())

	_, _, err = peerConn.ReadMessage()
	assert.Error(t, err, "После ухода host соединение с peer должно закрыться")
}
