package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/pkg/errors"

	"github.com/vpbuyanov/syncplay/internal/gen"
)

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	// одну комнату хранит набор подключений и указатель на «хоста»
	rooms   = make(map[openapi_types.UUID]*RoomSession)
	roomsMu sync.Mutex
)

type RoomSession struct {
	Host    *websocket.Conn
	Peers   map[*websocket.Conn]bool
	Session sync.Mutex
}

type Message struct {
	Type    string          `json:"type"`    // "join","offer","answer","ice"
	Payload json.RawMessage `json:"payload"` // SDP или ICE
}

func (s *Server) ConnectRoomWS(c echo.Context, roomID openapi_types.UUID) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Detail: "something wrong",
		})
	}
	defer func(ws *websocket.Conn) {
		err = ws.Close()
		if err != nil {
			slog.Error("Err", "err", err)
		}
	}(ws)

	// получаем или создаём RoomSession
	roomsMu.Lock()
	rs, ok := rooms[roomID]
	if !ok {
		rs = &RoomSession{Peers: make(map[*websocket.Conn]bool)}
		rooms[roomID] = rs
	}
	roomsMu.Unlock()

	// теперь добавляем этого ws как host или peer
	rs.Session.Lock()
	if rs.Host == nil {
		rs.Host = ws
		// можно уведомить клиента, что он host
		if err = ws.WriteJSON(map[string]string{"type": "role", "role": "host"}); err != nil {
			return errors.Wrap(err, "write json")
		}
	} else {
		rs.Peers[ws] = true
		if err = ws.WriteJSON(map[string]string{"type": "role", "role": "peer"}); err != nil {
			return errors.Wrap(err, "write json")
		}
	}
	rs.Session.Unlock()

	// читаем и ретранслируем сигналы
	for {
		var msg Message
		if err = ws.ReadJSON(&msg); err != nil {
			break
		}

		rs.Session.Lock()
		// если от host-а — отправляем всем peers
		if ws == rs.Host {
			for peer := range rs.Peers {
				if err = peer.WriteJSON(msg); err != nil {
					return errors.Wrap(err, "write json")
				}
			}
		} else if rs.Host != nil {
			if err = rs.Host.WriteJSON(msg); err != nil {
				return errors.Wrap(err, "write json")
			}
		}
		rs.Session.Unlock()
	}

	// при выходе чистим
	rs.Session.Lock()
	if ws == rs.Host {
		// host уходит — продропать комнату целиком
		for peer := range rs.Peers {
			if err = peer.Close(); err != nil {
				return errors.Wrap(err, "peer close")
			}
		}
		roomsMu.Lock()
		delete(rooms, roomID)
		roomsMu.Unlock()
	} else {
		delete(rs.Peers, ws)
	}
	rs.Session.Unlock()

	return nil
}
