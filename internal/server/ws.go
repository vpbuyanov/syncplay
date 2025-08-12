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

// maybeDeleteRoom безопасно удаляет комнату из глобальной карты,
// делая двойную проверку под roomsMu -> sess.Session.
func maybeDeleteRoom(roomID openapi_types.UUID, sess *roomSession) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	// Комната могла быть уже удалена/заменена
	cur := rooms[roomID]
	if cur != sess {
		return
	}

	sess.Session.Lock()
	defer sess.Session.Unlock()

	if len(sess.Peers) == 0 {
		delete(rooms, roomID)
	}
}

func (s *Server) ConnectRoomWS(c echo.Context, roomID openapi_types.UUID) error {
	// Проверяем, что комната существует в БД до апгрейда
	exists, err := s.m.RoomExistsUUID(c.Request().Context(), roomID)
	if err != nil {
		c.Logger().Errorf("RoomExistsUUID err: %v", err)
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Detail: "db error"})
	}
	if !exists {
		return c.JSON(http.StatusNotFound, gen.ErrorResponse{Detail: "room not found"})
	}

	// Upgrade до WebSocket
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

	// Снимаем слепки существующих и получателей "new-peer"
	sess.Session.Lock()

	existing := make([]string, 0, len(sess.Peers))
	for id := range sess.Peers {
		existing = append(existing, id)
	}

	// Добавляем себя
	sess.Peers[peerID] = ws

	// Получатели "new-peer" (все, кроме нас)
	recipients := make([]*websocket.Conn, 0, len(sess.Peers)-1)
	for id, pc := range sess.Peers {
		if id != peerID {
			recipients = append(recipients, pc)
		}
	}

	sess.Session.Unlock()

	// Приветствие нового
	if err = ws.WriteJSON(message{Type: "welcome", ID: peerID}); err != nil {
		return err
	}
	if err = ws.WriteJSON(message{Type: "existing-peers", Peers: existing}); err != nil {
		return err
	}
	for _, pc := range recipients {
		if err = pc.WriteJSON(message{Type: "new-peer", ID: peerID}); err != nil {
			c.Logger().Errorf("failed to send 'new-peer' (roomID=%s, newPeer=%s): %v", roomID, peerID, err)
		}
	}

	// Основной цикл сигналинга
	for {
		var msg message
		if err = ws.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type != "signal" || msg.To == "" {
			continue
		}

		// Берём ссылку на получателя под локом сессии
		var dest *websocket.Conn
		sess.Session.Lock()
		dest = sess.Peers[msg.To]
		sess.Session.Unlock()

		// Пишем уже без лока (ошибки логируем)
		if dest != nil {
			if err = dest.WriteJSON(message{
				Type:    "signal",
				From:    peerID,
				To:      msg.To,
				Payload: msg.Payload,
			}); err != nil {
				c.Logger().Errorf("failed to forward signal: roomID=%s from=%s to=%s: %v", roomID, peerID, msg.To, err)
			}
		}
	}

	// Клиент уходит: удаляем из Peers и шлём 'peer-left' остальным
	var leftRecipients []*websocket.Conn
	sess.Session.Lock()

	delete(sess.Peers, peerID)

	leftRecipients = make([]*websocket.Conn, 0, len(sess.Peers))
	for _, pc := range sess.Peers {
		leftRecipients = append(leftRecipients, pc)
	}

	sess.Session.Unlock()

	for _, pc := range leftRecipients {
		if err = pc.WriteJSON(message{Type: "peer-left", ID: peerID}); err != nil {
			c.Logger().Errorf("failed to notify 'peer-left' (roomID=%s, leftPeer=%s): %v", roomID, peerID, err)
		}
	}

	// Безопасная попытка удалить комнату, если она опустела
	maybeDeleteRoom(roomID, sess)

	return nil
}
