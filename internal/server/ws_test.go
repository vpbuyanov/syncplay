package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/mock/gomock"

	"github.com/vpbuyanov/syncplay/internal/gen"
)

// дедлайн на чтение, чтобы тесты не висели
func readJSONWithTimeout(t *testing.T, conn *websocket.Conn, v interface{}) error {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	return conn.ReadJSON(v)
}

// сброс глобального состояния комнат
func clearRooms() {
	roomsMu.Lock()
	rooms = make(map[openapi_types.UUID]*roomSession)
	roomsMu.Unlock()
}

func TestConnectRoomWS_TwoPeersSignalAndLeave_ModelMock(t *testing.T) {
	clearRooms()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockModel := NewMockmodelRoom(ctrl)
	srv := &Server{m: mockModel}

	e := echo.New()
	e.GET("/ws/:roomID", func(c echo.Context) error {
		rid := c.Param("roomID")
		u, err := uuid.Parse(rid)
		if err != nil {
			return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Detail: "invalid room ID"})
		}
		return srv.ConnectRoomWS(c, u)
	})

	ts := httptest.NewServer(e)
	defer ts.Close()

	roomID := uuid.New().String()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/" + roomID

	// Комната существует для обоих подключений
	mockModel.
		EXPECT().
		RoomExistsUUID(gomock.Any(), gomock.AssignableToTypeOf(openapi_types.UUID{})).
		Return(true, nil).
		Times(2)

	// peer1
	peer1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("peer1 dial failed: %v", err)
	}
	defer peer1.Close()

	var w1 message
	if err = readJSONWithTimeout(t, peer1, &w1); err != nil {
		t.Fatalf("peer1 welcome read: %v", err)
	}
	if w1.Type != "welcome" || w1.ID == "" {
		t.Fatalf("unexpected welcome1: %+v", w1)
	}
	p1 := w1.ID

	var ex1 message
	if err = readJSONWithTimeout(t, peer1, &ex1); err != nil {
		t.Fatalf("peer1 existing read: %v", err)
	}
	if ex1.Type != "existing-peers" || len(ex1.Peers) != 0 {
		t.Fatalf("expected no existing peers, got: %+v", ex1)
	}

	// peer2
	peer2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("peer2 dial failed: %v", err)
	}
	defer peer2.Close()

	var w2 message
	if err = readJSONWithTimeout(t, peer2, &w2); err != nil {
		t.Fatalf("peer2 welcome read: %v", err)
	}
	if w2.Type != "welcome" || w2.ID == "" {
		t.Fatalf("unexpected welcome2: %+v", w2)
	}
	p2 := w2.ID

	var ex2 message
	if err = readJSONWithTimeout(t, peer2, &ex2); err != nil {
		t.Fatalf("peer2 existing read: %v", err)
	}
	if ex2.Type != "existing-peers" || len(ex2.Peers) != 1 || ex2.Peers[0] != p1 {
		t.Fatalf("peer2 existing mismatch: %+v", ex2)
	}

	// peer1 должен получить new-peer(p2)
	var np message
	if err = readJSONWithTimeout(t, peer1, &np); err != nil {
		t.Fatalf("peer1 new-peer read: %v", err)
	}
	if np.Type != "new-peer" || np.ID != p2 {
		t.Fatalf("unexpected new-peer: %+v", np)
	}

	// сигнал p1 -> p2
	payload := json.RawMessage(`{"hello":"world"}`)
	if err = peer1.WriteJSON(message{Type: "signal", To: p2, Payload: payload}); err != nil {
		t.Fatalf("peer1 send signal: %v", err)
	}

	var recv message
	if err = readJSONWithTimeout(t, peer2, &recv); err != nil {
		t.Fatalf("peer2 recv signal: %v", err)
	}
	if recv.Type != "signal" || recv.From != p1 || string(recv.Payload) != string(payload) {
		t.Fatalf("signal mismatch: %+v", recv)
	}

	// peer2 уходит -> peer1 получает peer-left
	_ = peer2.Close()
	var left message
	if err := readJSONWithTimeout(t, peer1, &left); err != nil {
		t.Fatalf("peer1 read peer-left: %v", err)
	}
	if left.Type != "peer-left" || left.ID != p2 {
		t.Fatalf("unexpected peer-left: %+v", left)
	}

	_ = peer1.Close()

	time.Sleep(100 * time.Millisecond)
	roomsMu.Lock()
	defer roomsMu.Unlock()
	if len(rooms) != 0 {
		t.Fatalf("rooms not cleaned, len=%d", len(rooms))
	}
}

func TestConnectRoomWS_SignalToRandomPeer_DoesNotPanic_ModelMock(t *testing.T) {
	clearRooms()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockModel := NewMockmodelRoom(ctrl)
	srv := &Server{m: mockModel}

	e := echo.New()
	e.GET("/ws/:roomID", func(c echo.Context) error {
		rid := c.Param("roomID")
		u, err := uuid.Parse(rid)
		if err != nil {
			return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Detail: "invalid room ID"})
		}
		return srv.ConnectRoomWS(c, u)
	})

	ts := httptest.NewServer(e)
	defer ts.Close()

	roomID := uuid.New().String()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/" + roomID

	// Комната существует
	mockModel.
		EXPECT().
		RoomExistsUUID(gomock.Any(), gomock.AssignableToTypeOf(openapi_types.UUID{})).
		Return(true, nil)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	var w message
	if err = readJSONWithTimeout(t, conn, &w); err != nil {
		t.Fatalf("welcome read: %v", err)
	}
	var ex message
	if err = readJSONWithTimeout(t, conn, &ex); err != nil {
		t.Fatalf("existing read: %v", err)
	}

	// сигнал на несуществующий peer — соединение не должно падать
	random := uuid.NewString()
	if err = conn.WriteJSON(message{Type: "signal", To: random, Payload: json.RawMessage(`{"x":1}`)}); err != nil {
		t.Fatalf("send bad signal: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err = conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		t.Fatalf("connection closed unexpectedly: %v", err)
	}
}

func TestConnectRoomWS_RoomNotFound_BeforeUpgrade(t *testing.T) {
	clearRooms()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockModel := NewMockmodelRoom(ctrl)
	srv := &Server{m: mockModel}

	e := echo.New()
	e.GET("/ws/:roomID", func(c echo.Context) error {
		rid := c.Param("roomID")
		u, err := uuid.Parse(rid)
		if err != nil {
			return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Detail: "invalid room ID"})
		}
		return srv.ConnectRoomWS(c, u)
	})

	ts := httptest.NewServer(e)
	defer ts.Close()

	roomID := uuid.New().String()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/" + roomID

	// Комната НЕ существует
	mockModel.
		EXPECT().
		RoomExistsUUID(gomock.Any(), gomock.AssignableToTypeOf(openapi_types.UUID{})).
		Return(false, nil)

	// Dial не получит 101 Switching Protocols → вернётся ошибка bad handshake.
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected bad handshake error, got nil")
	}
	if resp == nil {
		t.Fatalf("expected HTTP response, got nil err=%v", err)
	}
	// ожидаем 404 (или 400 — подстрой под свой ConnectRoomWS)
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
}
