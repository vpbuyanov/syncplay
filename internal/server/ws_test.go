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
	"github.com/vpbuyanov/syncplay/internal/gen"
)

// readJSONWithTimeout wraps ReadJSON with a deadline to avoid hanging tests.
func readJSONWithTimeout(t *testing.T, conn *websocket.Conn, v interface{}) error {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	if err := conn.SetReadDeadline(deadline); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	return conn.ReadJSON(v)
}

// clearRooms resets the global state so tests don't interfere with each other.
func clearRooms() {
	roomsMu.Lock()
	rooms = make(map[openapi_types.UUID]*roomSession)
	roomsMu.Unlock()
}

func TestConnectRoomWS_TwoPeersSignalAndLeave(t *testing.T) {
	clearRooms()
	e := echo.New()
	e.GET("/ws/:roomID", func(c echo.Context) error {
		roomIDStr := c.Param("roomID")
		parsed, err := uuid.Parse(roomIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Detail: "invalid room ID"})
		}
		roomID := openapi_types.UUID(parsed)
		return (&Server{}).ConnectRoomWS(c, roomID)
	})

	ts := httptest.NewServer(e)
	defer ts.Close()

	// Генерируем UUID комнаты.
	roomUUID := uuid.New()
	roomIDStr := roomUUID.String()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/" + roomIDStr

	// Подключаем первого peer-а.
	peer1Conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("peer1 dial failed: %v", err)
	}
	defer peer1Conn.Close()

	var welcome1 message
	if err := readJSONWithTimeout(t, peer1Conn, &welcome1); err != nil {
		t.Fatalf("peer1 failed to read welcome: %v", err)
	}
	if welcome1.Type != "welcome" || welcome1.ID == "" {
		t.Fatalf("unexpected welcome1: %+v", welcome1)
	}
	peer1ID := welcome1.ID

	var existing1 message
	if err := readJSONWithTimeout(t, peer1Conn, &existing1); err != nil {
		t.Fatalf("peer1 failed to read existing-peers: %v", err)
	}
	if existing1.Type != "existing-peers" || len(existing1.Peers) != 0 {
		t.Fatalf("expected no existing peers for peer1, got: %+v", existing1)
	}

	// Подключаем второго peer-а.
	peer2Conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("peer2 dial failed: %v", err)
	}
	defer peer2Conn.Close()

	var welcome2 message
	if err := readJSONWithTimeout(t, peer2Conn, &welcome2); err != nil {
		t.Fatalf("peer2 failed to read welcome: %v", err)
	}
	if welcome2.Type != "welcome" || welcome2.ID == "" {
		t.Fatalf("unexpected welcome2: %+v", welcome2)
	}
	peer2ID := welcome2.ID

	var existing2 message
	if err := readJSONWithTimeout(t, peer2Conn, &existing2); err != nil {
		t.Fatalf("peer2 failed to read existing-peers: %v", err)
	}
	if existing2.Type != "existing-peers" {
		t.Fatalf("expected existing-peers for peer2, got: %+v", existing2)
	}
	if len(existing2.Peers) != 1 || existing2.Peers[0] != peer1ID {
		t.Fatalf("peer2 existing peers mismatch: %+v", existing2)
	}

	// Первый peer должен получить new-peer про второго.
	var newPeerMsg message
	if err := readJSONWithTimeout(t, peer1Conn, &newPeerMsg); err != nil {
		t.Fatalf("peer1 failed to read new-peer: %v", err)
	}
	if newPeerMsg.Type != "new-peer" || newPeerMsg.ID != peer2ID {
		t.Fatalf("unexpected new-peer for peer1: %+v", newPeerMsg)
	}

	// Проверяем сигнал от первого ко второму.
	payload := json.RawMessage(`{"hello":"world"}`)
	signalMsg := message{Type: "signal", To: peer2ID, Payload: payload}
	if err := peer1Conn.WriteJSON(signalMsg); err != nil {
		t.Fatalf("peer1 failed to send signal: %v", err)
	}

	var receivedSignal message
	if err := readJSONWithTimeout(t, peer2Conn, &receivedSignal); err != nil {
		t.Fatalf("peer2 failed to receive signal: %v", err)
	}
	if receivedSignal.Type != "signal" || receivedSignal.From != peer1ID || string(receivedSignal.Payload) != string(payload) {
		t.Fatalf("signal mismatch: %+v", receivedSignal)
	}

	// Закрываем второго и ждём уведомление peer-left на первом.
	if err := peer2Conn.Close(); err != nil {
		t.Fatalf("failed to close peer2: %v", err)
	}

	var leftMsg message
	if err := readJSONWithTimeout(t, peer1Conn, &leftMsg); err != nil {
		t.Fatalf("peer1 failed to read peer-left: %v", err)
	}
	if leftMsg.Type != "peer-left" || leftMsg.ID != peer2ID {
		t.Fatalf("unexpected peer-left: %+v", leftMsg)
	}

	// Закрываем первого.
	if err := peer1Conn.Close(); err != nil {
		t.Fatalf("failed to close peer1: %v", err)
	}

	// Даём серверу немного времени, чтобы почистить комнату.
	time.Sleep(100 * time.Millisecond)

	roomsMu.Lock()
	if len(rooms) != 0 {
		t.Fatalf("expected rooms map to be empty after all peers left, got %d", len(rooms))
	}
	roomsMu.Unlock()
}

func TestConnectRoomWS_SignalToNonexistentPeerDoesNotPanic(t *testing.T) {
	clearRooms()
	e := echo.New()
	e.GET("/ws/:roomID", func(c echo.Context) error {
		roomIDStr := c.Param("roomID")
		parsed, err := uuid.Parse(roomIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Detail: "invalid room ID"})
		}
		roomID := openapi_types.UUID(parsed)
		return (&Server{}).ConnectRoomWS(c, roomID)
	})

	ts := httptest.NewServer(e)
	defer ts.Close()

	roomUUID := uuid.New()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/" + roomUUID.String()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	var welcome message
	if err := readJSONWithTimeout(t, conn, &welcome); err != nil {
		t.Fatalf("failed to read welcome: %v", err)
	}
	var existing message
	if err := readJSONWithTimeout(t, conn, &existing); err != nil {
		t.Fatalf("failed to read existing-peers: %v", err)
	}

	// Посылаем сигнал на несуществующий peer, не должно приводить к падению.
	randomID := uuid.NewString()
	badSignal := message{Type: "signal", To: randomID, Payload: json.RawMessage(`{"x":1}`)}
	if err := conn.WriteJSON(badSignal); err != nil {
		t.Fatalf("failed to send bad signal: %v", err)
	}

	// Небольшая пауза, затем пингуем чтобы убедиться, что соединение живо.
	time.Sleep(50 * time.Millisecond)
	if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		t.Fatalf("connection closed unexpectedly after bad signal: %v", err)
	}
}
