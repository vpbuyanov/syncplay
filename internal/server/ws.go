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

func (s *Server) ConnectRoomWS(c echo.Context, roomID openapi_types.UUID) error {
	exists, err := s.m.RoomExistsUUID(c.Request().Context(), roomID)
	if err != nil {
		c.Logger().Errorf("RoomExistsUUID err: %v", err)
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Detail: "db error"})
	}
	if !exists {
		return c.JSON(http.StatusNotFound, gen.ErrorResponse{Detail: "room not found"})
	}

	// Upgrade
	ws, err := upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Detail: "ws upgrade failed"})
	}
	defer ws.Close()

	peerID := uuid.NewString()

	// Получаем/создаём сессию комнаты (под глобальным локом)
	roomsMu.Lock()
	sess, ok := rooms[roomID]
	if !ok {
		sess = &roomSession{Peers: make(map[string]*websocket.Conn)}
		rooms[roomID] = sess
	}
	roomsMu.Unlock()

	var existing []string
	var recipients []*websocket.Conn

	sess.Session.Lock()
	existing = make([]string, 0, len(sess.Peers))
	for id := range sess.Peers {
		existing = append(existing, id)
	}

	// добавляем себя
	sess.Peers[peerID] = ws

	// получатели "new-peer" (все, кроме нас)
	recipients = make([]*websocket.Conn, 0, len(sess.Peers)-1)
	for id, pc := range sess.Peers {
		if id != peerID {
			recipients = append(recipients, pc)
		}
	}
	sess.Session.Unlock()

	if err = ws.WriteJSON(message{Type: "welcome", ID: peerID}); err != nil {
		return err
	}
	if err = ws.WriteJSON(message{Type: "existing-peers", Peers: existing}); err != nil {
		return err
	}
	for _, pc := range recipients {
		_ = pc.WriteJSON(message{Type: "new-peer", ID: peerID})
	}

	for {
		var msg message
		if err := ws.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type != "signal" || msg.To == "" {
			continue
		}

		var dest *websocket.Conn
		sess.Session.Lock()
		dest = sess.Peers[msg.To]
		sess.Session.Unlock()

		if dest != nil {
			_ = dest.WriteJSON(message{
				Type:    "signal",
				From:    peerID,
				To:      msg.To,
				Payload: msg.Payload,
			})
		}
	}

	var leftRecipients []*websocket.Conn
	sess.Session.Lock()
	delete(sess.Peers, peerID)
	leftRecipients = make([]*websocket.Conn, 0, len(sess.Peers))
	for _, pc := range sess.Peers {
		leftRecipients = append(leftRecipients, pc)
	}
	empty := len(sess.Peers) == 0
	sess.Session.Unlock()

	for _, pc := range leftRecipients {
		_ = pc.WriteJSON(message{Type: "peer-left", ID: peerID})
	}

	if empty {
		roomsMu.Lock()
		delete(rooms, roomID)
		roomsMu.Unlock()
	}

	return nil
}
