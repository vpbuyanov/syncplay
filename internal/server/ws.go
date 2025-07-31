package server

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/vpbuyanov/syncplay/internal/gen"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	rooms   = make(map[openapi_types.UUID]*roomSession)
	roomsMu sync.Mutex
)

type roomSession struct {
	Peers   map[string]*websocket.Conn
	Session sync.Mutex
}

type message struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Peers   []string        `json:"peers,omitempty"`
	From    string          `json:"from,omitempty"`
	To      string          `json:"to,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ConnectRoomWS — мультипиринговая версия WS
func (s *Server) ConnectRoomWS(c echo.Context, roomID openapi_types.UUID) error {
	// 1) Upgrade
	ws, err := upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Detail: "ws upgrade failed"})
	}
	defer ws.Close()

	// 2) Генерация peerID
	peerID := uuid.NewString()

	// 3) Получаем/создаём сессию
	roomsMu.Lock()
	sess, ok := rooms[roomID]
	if !ok {
		sess = &roomSession{Peers: make(map[string]*websocket.Conn)}
		rooms[roomID] = sess
	}
	roomsMu.Unlock()

	// 4) Сбор существующих peer'ов
	sess.Session.Lock()
	existing := make([]string, 0, len(sess.Peers))
	for id := range sess.Peers {
		existing = append(existing, id)
	}
	sess.Peers[peerID] = ws
	sess.Session.Unlock()

	// 5) Приветствие нового
	if err = ws.WriteJSON(message{Type: "welcome", ID: peerID}); err != nil {
		//nolint:wrapcheck
		return err
	}

	if err = ws.WriteJSON(message{Type: "existing-peers", Peers: existing}); err != nil {
		//nolint:wrapcheck
		return err
	}

	// 6) Уведомляем всех старых о new-peer
	sess.Session.Lock()
	for id, peerConn := range sess.Peers {
		if id == peerID {
			continue
		}
		if err = peerConn.WriteJSON(message{Type: "new-peer", ID: peerID}); err != nil {
			//nolint:wrapcheck
			return err
		}
	}
	sess.Session.Unlock()

	// 7) Чтение-сигналинг
	for {
		var msg message
		if err := ws.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == "signal" && msg.To != "" {
			sess.Session.Lock()
			if dest, ok := sess.Peers[msg.To]; ok {
				if err = dest.WriteJSON(message{
					Type:    "signal",
					From:    peerID,
					To:      msg.To,
					Payload: msg.Payload,
				}); err != nil {
					//nolint:wrapcheck
					return err
				}
			}
			sess.Session.Unlock()
		}
	}

	// 8) Удаление при выходе
	sess.Session.Lock()
	delete(sess.Peers, peerID)
	for _, peerConn := range sess.Peers {
		if err = peerConn.WriteJSON(message{Type: "peer-left", ID: peerID}); err != nil {
			//nolint:wrapcheck
			return err
		}
	}
	sess.Session.Unlock()

	// 9) Если комната пуста — чистим
	roomsMu.Lock()
	if len(sess.Peers) == 0 {
		delete(rooms, roomID)
	}
	roomsMu.Unlock()

	return nil
}
